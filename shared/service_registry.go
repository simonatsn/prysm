// Package shared includes useful utilities globally accessible in
// the Prysm monorepo.
package shared

import (
	"context"
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "registry")

// Service is a struct that can be registered into a ServiceRegistry for
// easy dependency management.
type Service interface {
	// Start spawns any goroutines required by the service.
	Start(ctx context.Context)
	// Stop terminates all goroutines belonging to the service,
	// blocking until they are all terminated.
	Stop(ctx context.Context) error
	// Status returns error if the service is not considered healthy.
	Status() error
}

// ServiceContext represents a cancellable context that should be used when interacting with the service.
// It helps manage the service's lifetime by providing a cancellation mechanism.
type ServiceContext struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

// ServiceRegistry provides a useful pattern for managing services.
// It allows for ease of dependency management and ensures services
// dependent on others use the same references in memory.
type ServiceRegistry struct {
	contexts     map[reflect.Type]*ServiceContext // map of types to contexts that can be cancelled.
	services     map[reflect.Type]Service         // map of types to services.
	serviceTypes []reflect.Type                   // keep an ordered slice of registered service types.
}

// NewServiceRegistry starts a registry instance for convenience
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		contexts: make(map[reflect.Type]*ServiceContext),
		services: make(map[reflect.Type]Service),
	}
}

// NewServiceContext returns a fresh service context
func (s *ServiceRegistry) NewServiceContext() *ServiceContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &ServiceContext{Ctx: ctx, Cancel: cancel}
}

// StartAll initialized each service in order of registration.
func (s *ServiceRegistry) StartAll() {
	log.Debugf("Starting %d services: %v", len(s.serviceTypes), s.serviceTypes)
	for _, kind := range s.serviceTypes {
		log.Debugf("Starting service type %v", kind)
		go s.services[kind].Start(s.contexts[kind].Ctx)
	}
}

// StopAll ends every service in reverse order of registration, logging a
// panic if any of them fail to stop.
func (s *ServiceRegistry) StopAll() {
	for i := len(s.serviceTypes) - 1; i >= 0; i-- {
		kind := s.serviceTypes[i]
		ctx := s.contexts[kind]
		service := s.services[kind]
		if err := service.Stop(ctx.Ctx); err != nil {
			log.WithError(err).Errorf("Could not stop the following service: %v, %v", kind, err)
		}
		ctx.Cancel()
	}
}

// Statuses returns a map of Service type -> error. The map will be populated
// with the results of each service.Status() method call.
func (s *ServiceRegistry) Statuses() map[reflect.Type]error {
	m := make(map[reflect.Type]error, len(s.serviceTypes))
	for _, kind := range s.serviceTypes {
		m[kind] = s.services[kind].Status()
	}
	return m
}

// RegisterService appends a service constructor function to the service
// registry.
func (s *ServiceRegistry) RegisterService(service Service, serviceCtx *ServiceContext) error {
	kind := reflect.TypeOf(service)
	if _, exists := s.services[kind]; exists {
		return fmt.Errorf("service already exists: %v", kind)
	}
	s.contexts[kind] = serviceCtx
	s.services[kind] = service
	s.serviceTypes = append(s.serviceTypes, kind)
	return nil
}

// FetchService takes in a struct pointer and sets the value of that pointer
// to a service currently stored in the service registry. This ensures the input argument is
// set to the right pointer that refers to the originally registered service.
func (s *ServiceRegistry) FetchService(service interface{}) error {
	if reflect.TypeOf(service).Kind() != reflect.Ptr {
		return fmt.Errorf("input must be of pointer type, received value type instead: %T", service)
	}
	element := reflect.ValueOf(service).Elem()
	if running, ok := s.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return fmt.Errorf("unknown service: %T", service)
}
