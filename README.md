# byos-screen-renderer

Simple app to render a TRMNL screen

## Local dev
```
go run main.go
```

## Build and run docker
```
docker build -t byos-screen-renderer .
docker run -v "$(pwd)/tmp:/output" -p 8080:8080 --rm byos-screen-renderer
```

## Usage
```
curl -X POST http://localhost:8080/render/myslug \
  -H "Content-Type: application/json" \
  -d '	{
		"template": "<body>hi</body>",
		"title": "App running",
		"date": "Jul 26, 2025",
		"column_1_title": "Nested Liquid Example with Array 1",
		"column_2_title": "Nested Liquid Example with Array 2",
		"column_1": {
			"Name":  "Eric",
			"email": "eric@example.com"
		},
		"column_2": {
			"Size":  "5",
			"Type": "cool"
		}
	}'
```

Note, the template field is optional. By default, template.liquid is used.

To fetch the results:
```
curl http://localhost:8080/screenshot.png/myslug -o output.png
```


## Next feature ideas

- [x] passing key on render that must be passed to retrieve the image (query param handle)
- [x] split out liquid layout from body template, allow template per handle
- [x] cache rendered files, leverage docker dir mapping to support application reboots
