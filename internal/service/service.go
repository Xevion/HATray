package service

// This is an intentionally very-simple interface as the main program entrypoint needs to know very little about the service layer.
// The service layer is completely responsible for the lifecycle of the application, implemented per-platform.
type Service interface {
	Run() error
}

// You create a service using the NewService() function, implemented per-platform. If you don't have a NewService() function, you can't create a service on your platform.
