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
3. **Run the API server:**
   ```sh
   go run main.go
   ```
   The server will start on port 8080 by default. You can set the `PORT` environment variable to change the port.

> **Note:** Redis caching is not yet implemented. The codebase is structured to allow easy integration of Redis in the future.

## Usage
- **Get current weather:**
  ```