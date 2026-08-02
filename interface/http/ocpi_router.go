package http

import (
	"net/http"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/handlers"

	"github.com/go-chi/chi/v5"
)

// OCPIDependencies bundles what the OCPI routes need.
type OCPIDependencies struct {
	Partners *services.OCPIPartnerService
	Events   *services.OCPIEventService
	Commands *services.OCPICommandService
	// Settings supplies the live form defaults edited in the Config tab.
	Settings *services.AppSettingsService
	// BasicAuth wraps the control API. Nil means no authentication.
	BasicAuth func(http.Handler) http.Handler
}

// RegisterOCPIRoutes registers the partner control API and the partner
// receiver endpoints.
//
// The control API under /ocpi/api is protected by basic auth. The receiver
// routes under /ocpi/p are deliberately exempt: our platform calls them with
// the partner's OCPI `Token` credential instead.
func RegisterOCPIRoutes(router *chi.Mux, deps OCPIDependencies) {
	partnerHandler := handlers.NewOCPIPartnerHandler(deps.Partners, deps.Events, deps.Commands, deps.Settings)
	receiverHandler := handlers.NewOCPIReceiverHandler(deps.Partners, deps.Events)

	router.Route("/ocpi/api", func(r chi.Router) {
		if deps.BasicAuth != nil {
			r.Use(deps.BasicAuth)
		}

		r.Get("/defaults", partnerHandler.GetDefaults)

		r.Route("/partners", func(r chi.Router) {
			r.Get("/", partnerHandler.ListPartners)
			r.Post("/", partnerHandler.CreatePartner)

			r.Route("/{slug}", func(r chi.Router) {
				r.Get("/", partnerHandler.GetPartner)
				r.Put("/", partnerHandler.UpdatePartner)
				r.Delete("/", partnerHandler.DeletePartner)

				r.Post("/commands/start", partnerHandler.StartSession)
				r.Post("/commands/stop", partnerHandler.StopSession)

				r.Get("/events", partnerHandler.GetEvents)
				r.Delete("/events", partnerHandler.ClearEvents)
			})
		})
	})

	router.Route("/ocpi/p/{slug}", func(r chi.Router) {
		r.Route("/receiver/2.2.1", func(r chi.Router) {
			r.Put("/locations/*", receiverHandler.HandlePush(entities.OCPIEventLocation))
			r.Patch("/locations/*", receiverHandler.HandlePush(entities.OCPIEventLocation))

			r.Put("/tariffs/*", receiverHandler.HandlePush(entities.OCPIEventTariff))
			r.Delete("/tariffs/*", receiverHandler.HandlePush(entities.OCPIEventTariff))

			r.Put("/sessions/*", receiverHandler.HandlePush(entities.OCPIEventSession))
			r.Patch("/sessions/*", receiverHandler.HandlePush(entities.OCPIEventSession))

			r.Post("/cdrs", receiverHandler.HandlePush(entities.OCPIEventCDR))
		})

		r.Post("/commands/{commandType}/{uid}", receiverHandler.HandleCommandResult)
	})
}
