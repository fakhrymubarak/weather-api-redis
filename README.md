# Weather API

This is a challenge project from [Roadmap.sh](https://roadmap.sh/projects/weather-api-wrapper-service).

## Overview
A simple Weather API service that fetches weather data from external providers and caches responses using Redis for improved performance. This project is designed as a learning exercise to demonstrate API design, caching strategies, and integration with third-party services.

## Architecture Diagram
![Architecture Diagram](assets/arch-diagrams.png)

## Features
- Fetch current weather data for a given location
- Cache weather responses in Redis to reduce API calls
- Configurable cache expiration
- Simple RESTful API interface
- (Optional) Support for multiple weather providers

## Tech Stack
- Go (Golang)
- Redis
- (Optional) Docker
- (Optional) External Weather API (e.g., OpenWeatherMap, WeatherAPI)

## Setup
1. **Clone the repository:**
   ```sh
   git clone <repo-url>
   cd weather-api-redis
   ```
2. **Install dependencies:**
   ```sh
   # If using Go modules
   go mod tidy
   ```
3. **Configure environment variables:**
   - `REDIS_URL`: Redis connection string (e.g., `redis://localhost:6379`)
   - `WEATHER_API_KEY`: API key for the weather provider
   - (Optional) Other configuration as needed
4. **Run Redis:**
   - Make sure you have a Redis server running locally or use a cloud provider.
5. **Start the API server:**
   ```sh
   go run main.go
   ```

## Usage
- **Get current weather:**
  ```
  GET /weather?location=<city>
  ```
  **Response:**
  ```json
  {
    "location": "London",
    "temperature": 15.2,
    "description": "Partly cloudy",
    "cached": true
  }
  ```
- (Add more endpoints and examples as you implement them)

## Testing
- To run tests:
  ```sh
  go test ./...
  ```
- (Add more details as you add tests)

## Contributing
Contributions are welcome! Please open issues or submit pull requests for improvements or bug fixes.

## License
This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.