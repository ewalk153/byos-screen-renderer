FROM ruby:3.4-bookworm

ENV DEBIAN_FRONTEND=noninteractive

# System deps: Chromium + ImageMagick 6 + common headless libs + fonts
RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    imagemagick \
    ca-certificates \
    libasound2 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libatspi2.0-0 \
    libcairo2 \
    libcups2 \
    libdbus-1-3 \
    libdrm2 \
    libexpat1 \
    libgbm1 \
    libglib2.0-0 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libpango-1.0-0 \
    libpangocairo-1.0-0 \
    libx11-6 \
    libx11-xcb1 \
    libxcb1 \
    libxcomposite1 \
    libxcursor1 \
    libxdamage1 \
    libxext6 \
    libxfixes3 \
    libxi6 \
    libxrandr2 \
    libxrender1 \
    libxss1 \
    libxtst6 \
    fonts-dejavu fonts-noto-color-emoji \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install gems first for better layer caching
COPY Gemfile Gemfile.lock* ./
RUN bundle install --without development test --jobs 4 --retry 3

# App files
COPY . .

# App defaults (Chromium path inside Debian + output dir)
ENV CHROMIUM_PATH=/usr/bin/chromium
ENV OUTPUT_PATH=/output/

RUN mkdir -p /output

EXPOSE 8080
CMD ["bundle", "exec", "ruby", "server.rb"]
