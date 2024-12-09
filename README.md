# WebSocket Message Queue Server

## Summary

A robust and scalable WebSocket-based message queue server implemented in Go. This server facilitates real-time communication between clients through topics, allowing for message publishing and subscribing with reliable delivery guarantees. It incorporates efficient logging mechanisms, concurrency-safe operations, and comprehensive acknowledgment handling to ensure seamless message flow and system stability.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
  - [Server](#server)
  - [Client Management](#client-management)
  - [Topic Management](#topic-management)
  - [Message Acknowledgment](#message-acknowledgment)
  - [Logging](#logging)
- [Installation](#installation)
- [Usage](#usage)
- [Code Structure](#code-structure)
  - [Main Server (`main.go`)](#main-server-maingo)
  - [Topic Management (`topic.go`)](#topic-management-topicgo)
  - [Message Acknowledgment Manager (`manager.go`)](#message-acknowledgment-manager-managergo)
  - [Logger (`logger/logger.go`)](#logger-loggermanagergo)
- [Logging Details](#logging-details)
- [Contributing](#contributing)
- [License](#license)

## Overview

This project implements a WebSocket-based message queue server in Go, enabling clients to publish and subscribe to various topics. The server ensures reliable message delivery through acknowledgment mechanisms and manages multiple client connections concurrently. It leverages robust logging for monitoring and debugging purposes.

## Features

- **WebSocket Support**: Real-time bidirectional communication between clients and the server.
- **Topic-Based Messaging**: Clients can subscribe to and publish messages on specific topics.
- **Reliable Delivery**: Ensures messages are delivered and acknowledged to maintain consistency.
- **Concurrency Safe**: Handles multiple clients and topics simultaneously using synchronization primitives.
- **Structured Logging**: Utilizes Zap for high-performance logging with log rotation via Lumberjack.

## Architecture

### Server

The server manages client connections, topic subscriptions, message publishing, and acknowledgment handling. It employs Gorilla WebSocket for managing WebSocket connections and Zap for structured logging.

### Client Management

Each client connection is managed by a `ClientManager` instance, which handles subscriptions to topics, message acknowledgments, and communication with the server.

### Topic Management

Topics are managed by the `Topic` struct, which maintains a list of subscribers and a queue of messages. Clients can subscribe to multiple topics, and the server ensures that messages are delivered to all relevant subscribers.

### Message Acknowledgment

To guarantee reliable message delivery, the server implements acknowledgment mechanisms. Clients acknowledge received messages, and the server tracks these acknowledgments to manage message queues and retries in case of failures.

### Logging

The server uses the Zap logging library for structured and efficient logging. Logs are written to both the console and rotating log files, managed by Lumberjack, to ensure log files remain manageable in size and number.

## Installation

1. **Prerequisites**:

   - Go 1.16 or higher
   - Git

2. **Clone the Repository**:

   ```bash
   git clone https://github.com/yourusername/websocket-mq-server.git
   cd websocket-mq-server
   ```

3. **Install Dependencies**:

   ```bash
   go mod download
   ```

4. **Build the Server**:
   ```bash
   go build -o mq-server
   ```

## Usage

1. **Run the Server**:

   ```bash
   ./mq-server
   ```

   The server listens on port `8082` by default.

2. **Connect Clients**:
   Clients can connect to the server via WebSocket at `ws://localhost:8082/ws`.

3. **Subscribe to Topics**:
   Send a JSON message with the action `subscribe` and the desired topic.

4. **Publish Messages**:
   Send a JSON message with the action `publish`, the target topic, and the message data.

5. **Acknowledge Messages**:
   Clients should send acknowledgment messages for received messages to ensure reliable delivery.

## Code Structure

### Main Server (`main.go`)

Handles the initialization of the server, client connections, message processing, and routing.

#### Key Components:

- **Server**: Manages all client connections and topics.
- **ClientManager**: Manages individual client connections, subscriptions, and acknowledgment tracking.
- **Message Handling**: Processes incoming messages and routes them to appropriate handlers based on action types (`subscribe`, `publish`, `ack`).

### Topic Management (`topic.go`)

Represents a message topic, managing subscribers and the message queue.

#### Key Components:

- **Topic**: Maintains subscribers and queued messages.
- **Publish**: Adds messages to the topic queue and notifies subscribers.
- **Subscriber Management**: Adds or removes clients from the subscriber list.

### Message Acknowledgment Manager (`manager.go`)

Tracks acknowledgments from clients to ensure messages are delivered reliably.

#### Key Components:

- **MessageAckManager**: Maintains the last acknowledged message numbers per topic for each client.
- **Thread Safety**: Uses mutexes to protect concurrent access to acknowledgment maps.

### Logger (`logger/logger.go`)

Provides structured logging capabilities with log rotation.

#### Key Components:

- **Initialization**: Configures Zap logger with console and file encoders.
- **Log Rotation**: Utilizes Lumberjack for automatic log file rotation based on size, backups, and age.
- **Logging Functions**: Offers convenience functions (`Info`, `Debug`, `Warn`, `Error`) for logging at various levels.

## Logging Details

The server employs the Zap logging library for efficient and structured logging. Logs are output to both the console and a log file located at `./logs/app.log`. Log rotation is managed by Lumberjack with the following configuration:

- **Max Size**: 5 MB per log file before rotation.
- **Max Backups**: Keeps up to 3 old log files.
- **Max Age**: Retains log files for 28 days.
- **Compression**: Rotated log files are compressed to save space.

### Log Levels

- **Info**: General operational messages indicating the normal functioning of the server.
- **Debug**: Detailed information useful for debugging purposes.
- **Warn**: Indications of potential issues or unexpected events that do not halt the server.
- **Error**: Errors that have occurred, typically requiring attention.

## Contributing

Contributions are welcome! Please follow these steps to contribute:

1. **Fork the Repository**: Click the "Fork" button at the top of the repository page.

2. **Clone Your Fork**:

   ```bash
   git clone https://github.com/yourusername/websocket-mq-server.git
   cd websocket-mq-server
   ```

3. **Create a Branch**:

   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make Your Changes**: Implement your feature or fix.

5. **Commit Your Changes**:

   ```bash
   git commit -m "Add your detailed description here"
   ```

6. **Push to Your Fork**:

   ```bash
   git push origin feature/your-feature-name
   ```

7. **Submit a Pull Request**: Go to the original repository and submit a pull request with a detailed description of your changes.

## License

This project is licensed under the [MIT License](LICENSE).

---

Feel free to customize and expand upon this README to better fit the specific details and requirements of your project.
