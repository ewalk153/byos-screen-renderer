# frozen_string_literal: true
require "sinatra"
require "json"
require "liquid"
require "ferrum"
require "tempfile"
require "open3"
require "tmpdir"
require "fileutils"
require "uri"

set :bind, "0.0.0.0"
set :port, 8080
set :server, :puma

TEMPLATE_PATH = ENV.fetch("TEMPLATE_PATH", "template.liquid")
# Make OUTPUT_DIR absolute so we never depend on process CWD
OUTPUT_DIR    = File.expand_path(ENV.fetch("OUTPUT_PATH", ""), Dir.pwd) # e.g., "/output/"
OUTPUT_BASE   = "output.png"
MUTEX         = Mutex.new

helpers do
  def output_png_for(slug)
    FileUtils.mkdir_p(OUTPUT_DIR) if OUTPUT_DIR != "" # safe no-op if ""
    fname = [slug, OUTPUT_BASE].compact.join("-").sub(/^-/, "")
    OUTPUT_DIR == "" ? File.expand_path(fname, Dir.pwd) : File.join(OUTPUT_DIR, fname)
  end

  def chromium_path
    ENV["CHROMIUM_PATH"] # e.g., "/usr/bin/chromium" in Docker
  end

  def render_liquid(template_str, data)
    Liquid::Template.parse(template_str).render(data, strict_filters: false, strict_variables: false)
  end

  def read_template_from_request_or_file(data)
    if data.is_a?(Hash) && data["template"].is_a?(String) && !data["template"].empty?
      data["template"]
    else
      File.read(TEMPLATE_PATH)
    end
  end

  # Main worker
  def generate_screenshot(html, slug)
    # Absolute per-slug temp file path
    screenshot_tmp = File.join(Dir.tmpdir, "screenshot-#{slug.to_s.empty? ? 'default' : slug}.png")
    out_png        = output_png_for(slug)

    # 1) Write HTML to a temp file, render with Ferrum, capture bytes
    Tempfile.create(["render-", ".html"]) do |tmp|
      tmp.write(html)
      tmp.flush

      browser_opts = {
        headless: true,
        timeout: 20,
        browser_options: { "no-sandbox": nil, "disable-gpu": nil }
      }
      path = chromium_path
      browser_opts[:path] = path if path && !path.empty?

      browser = Ferrum::Browser.new(**browser_opts)
      page = browser.create_page
      page.go_to("file://#{tmp.path}")
      page.set_viewport(width: 800, height: 480, scale_factor: 1)
      browser.screenshot(path: screenshot_tmp)
      browser.quit
    end

    unless File.exist?(screenshot_tmp)
      warn "Screenshot file missing: #{screenshot_tmp}"
      raise "Screenshot file was not created"
    end

    # 2) Convert to 1-bit PNG using ImageMagick 7 `magick`
    convert_args = [
      "convert",
      screenshot_tmp,
      "-dither", "FloydSteinberg",
      "-remap", "pattern:gray50",
      "-depth", "1",
      "-strip",
      "png:#{out_png}"
    ]

    puts "INFO: converting temp=#{screenshot_tmp} -> out=#{out_png}"
    MUTEX.synchronize do
      _stdout, stderr, status = Open3.capture3(*convert_args)
      File.delete(screenshot_tmp) rescue nil
      unless status.success?
        warn "ImageMagick convert failed: #{stderr}"
        raise "ImageMagick convert failed"
      end
    end

    nil
  rescue => e
    warn "generate_screenshot error: #{e.class}: #{e.message}"
    warn "CHROMIUM_PATH=#{chromium_path.inspect}"
    warn "TMPDIR=#{Dir.tmpdir}, OUTPUT_DIR=#{OUTPUT_DIR}"
    raise
  end
end

get "/up" do
  "OK"
end

post "/render/:slug" do
  request.body.rewind
  data = JSON.parse(request.body.read) rescue nil
  halt 400, "Invalid JSON" unless data.is_a?(Hash)

  begin
    template_str = read_template_from_request_or_file(data)
  rescue => e
    halt 500, "Failed to read template: #{e.message}"
  end

  begin
    html = render_liquid(template_str, data)
  rescue => e
    halt 500, "Failed to render template: #{e.message}"
  end

  slug = params[:slug]

  Thread.new do
    begin
      generate_screenshot(html, slug)
    rescue => e
      warn "Screenshot generation failed: #{e.message}"
    end
  end

  status 202
  "Rendering started. Visit /screenshot.png/#{slug} to retrieve the result."
end

get "/screenshot.png/:slug" do
  path = output_png_for(params[:slug])
  halt 404, "Screenshot not ready" unless File.exist?(path)

  content_type "image/png"
  headers "Content-Length" => File.size(path).to_s
  File.binread(path)
end
