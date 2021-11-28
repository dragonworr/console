package api

import (
	"path/filepath"

	"github.com/cloudhut/common/middleware"
	"github.com/cloudhut/common/rest"
	"github.com/go-chi/chi"
	chimiddleware "github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// All the routes for the application are defined in one place.
func (api *API) routes() *chi.Mux {
	baseRouter := chi.NewRouter()
	baseRouter.NotFound(rest.HandleNotFound(api.Logger))
	baseRouter.MethodNotAllowed(rest.HandleMethodNotAllowed(api.Logger))

	instrument := middleware.NewInstrument(api.Cfg.MetricsNamespace)
	recoverer := middleware.Recoverer{Logger: api.Logger}
	handleBasePath := createHandleBasePathMiddleware(api.Cfg.REST.BasePath, api.Cfg.REST.SetBasePathFromXForwardedPrefix, api.Cfg.REST.StripPrefix)
	baseRouter.Use(recoverer.Wrap,
		chimiddleware.RealIP,
		// requirePrefix(api.Cfg.REST.BasePath), // only for debugging
		handleBasePath,
	)

	baseRouter.Group(func(router chi.Router) {
		// Init middlewares - Do set up of any shared/third-party middleware and handlers
		if api.Cfg.REST.CompressionLevel > 0 {
			api.Logger.Debug("using compression for all http routes", zap.Int("level", api.Cfg.REST.CompressionLevel))
			compressor := chimiddleware.NewCompressor(api.Cfg.REST.CompressionLevel)
			router.Use(compressor.Handler)
		}

		router.Use(
			middleware.Intercept,
			instrument.Wrap,
			// TODO: Add timeout middleware which allows route excludes
		)

		// This should be called here so that you can still add middlewares in the hook function.
		// Middlewares must be defined before routes.
		api.Hooks.Route.ConfigRouter(router)

		// Private routes - these should only be accessible from within Kubernetes or a protected ingress
		router.Group(func(r chi.Router) {
			r.Handle("/admin/metrics", promhttp.Handler())
			r.Handle("/admin/health", api.handleLivenessProbe())
			r.Handle("/admin/startup", api.handleStartupProbe())

			// Path must be prefixed with /debug otherwise it will be overridden, see: https://golang.org/pkg/net/http/pprof/
			r.Mount("/debug", chimiddleware.Profiler())
		})

		// API routes
		router.Group(func(r chi.Router) {
			r.Use(createSetVersionInfoHeader(api.version))
			api.Hooks.Route.ConfigAPIRouter(r)

			r.Route("/api", func(r chi.Router) {
				// Cluster
				r.Get("/api-versions", api.handleGetAPIVersions())
				r.Get("/brokers/{brokerID}/config", api.handleBrokerConfig())
				r.Get("/cluster", api.handleDescribeCluster())
				r.Get("/acls", api.handleGetACLsOverview())

				// Topics
				r.Get("/topics-configs", api.handleGetTopicsConfigs())
				r.Get("/topics-offsets", api.handleGetTopicsOffsets())
				r.Post("/topics-records", api.handlePublishTopicsRecords())
				r.Get("/topics", api.handleGetTopics())
				r.Delete("/topics/{topicName}", api.handleDeleteTopic())
				r.Delete("/topics/{topicName}/records", api.handleDeleteTopicRecords())
				r.Get("/topics/{topicName}/partitions", api.handleGetPartitions())
				r.Get("/topics/{topicName}/configuration", api.handleGetTopicConfig())
				r.Get("/topics/{topicName}/consumers", api.handleGetTopicConsumers())
				r.Get("/topics/{topicName}/documentation", api.handleGetTopicDocumentation())

				// Consumer Groups
				r.Get("/consumer-groups", api.handleGetConsumerGroups())
				r.Get("/consumer-groups/{groupId}", api.handleGetConsumerGroup())
				r.Patch("/consumer-groups/{groupId}", api.handlePatchConsumerGroup())
				r.Delete("/consumer-groups/{groupId}", api.handleDeleteConsumerGroupOffsets())

				// Bulk Operations
				r.Get("/operations/topic-details", api.handleGetAllTopicDetails())
				r.Get("/operations/reassign-partitions", api.handleGetPartitionReassignments())
				r.Patch("/operations/reassign-partitions", api.handlePatchPartitionAssignments())
				r.Patch("/operations/configs", api.handlePatchConfigs())

				// Schema Registry
				r.Get("/schemas", api.handleGetSchemaOverview())
				r.Get("/schemas/subjects/{subject}/versions/{version}", api.handleGetSchemaDetails())

				// Kafka Connect
				r.Get("/kafka-connect/connectors", api.handleGetConnectors())
				r.Get("/kafka-connect/clusters/{clusterName}", api.handleGetClusterInfo())
				r.Get("/kafka-connect/clusters/{clusterName}/connectors", api.handleGetClusterConnectors())
				r.Post("/kafka-connect/clusters/{clusterName}/connectors", api.handleCreateConnector())
				r.Get("/kafka-connect/clusters/{clusterName}/connectors/{connector}", api.handleGetConnector())
				r.Put("/kafka-connect/clusters/{clusterName}/connectors/{connector}", api.handlePutConnectorConfig())
				r.Put("/kafka-connect/clusters/{clusterName}/connector-plugins/{pluginClassName}/config/validate", api.handlePutValidateConnectorConfig())
				r.Delete("/kafka-connect/clusters/{clusterName}/connectors/{connector}", api.handleDeleteConnector())
				r.Put("/kafka-connect/clusters/{clusterName}/connectors/{connector}/pause", api.handlePauseConnector())
				r.Put("/kafka-connect/clusters/{clusterName}/connectors/{connector}/resume", api.handleResumeConnector())
				r.Post("/kafka-connect/clusters/{clusterName}/connectors/{connector}/restart", api.handleRestartConnector())
				r.Post("/kafka-connect/clusters/{clusterName}/connectors/{connector}/tasks/{taskID}/restart", api.handleRestartConnectorTask())

				// Kowl
				r.Get("/kowl/endpoints", api.handleGetEndpoints())
			})
		})

		if api.Cfg.ServeFrontend {
			// Check if the frontend directory 'build' exists
			frontendDir, err := filepath.Abs(api.Cfg.FrontendPath)
			if err != nil {
				api.Logger.Fatal("given frontend directory is invalid", zap.String("directory", frontendDir), zap.Error(err))
			}

			// SPA Files
			router.Group(func(r chi.Router) {
				handleIndex, handleResources := api.createFrontendHandlers(frontendDir)
				r.Get("/", handleIndex)
				r.Get("/*", handleResources)
			})
		} else {
			api.Logger.Info("no static files will be served as serving the frontend has been disabled")
		}
	})

	// Websockets live in it's own group because not all middlewares support websockets
	baseRouter.Group(func(wsRouter chi.Router) {
		api.Hooks.Route.ConfigWsRouter(wsRouter)

		wsRouter.Get("/api/topics/{topicName}/messages", api.handleGetMessages())
	})

	return baseRouter
}
