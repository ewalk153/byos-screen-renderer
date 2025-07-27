# Simple app to render a TRMNL screen

## Local dev
```
go run main.go
```

## Build and run docker
```
docker build -t go-liquid-renderer .
docker run -p 8080:8080 --rm go-liquid-renderer
```

## Usage
```
	curl -X POST http://localhost:8080/render \
  -H "Content-Type: application/json" \
  -d '	{
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
  and fetch the results

  ```
curl http://localhost:8080/screenshot.bmp -o output.bmp
  ```