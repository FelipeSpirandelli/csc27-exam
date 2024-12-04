package main

import (
	"time"

	"example.com/mq/logger"
	"go.uber.org/zap"
)

func main() {
	defer logger.Sync()

	logger.Info("Application started",
		zap.String("version", "1.0.0"),
		zap.Time("timestamp", time.Now()),
	)

	// Example of formatted message
	userID := 12345
	logger.Infof("User %d has logged in", userID)

	// Your application logic here
	doSomething()

	logger.Info("Application finished")
}

func doSomething() {
	logger.Debug("Doing something...")
	// Simulate work
	time.Sleep(2 * time.Second)
	logger.Debug("Done with something")
}
