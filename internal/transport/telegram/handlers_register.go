package telegram

// RegisterHandlers pairs every matcher with the handler that serves it. The
// registrar is an interface so the pairing itself stays testable: a matcher
// registered with the wrong handler would silently drop every update it
// matches.
func (d *Dispatcher) RegisterHandlers(registrar botRegistrar) {
	registrar.RegisterHandlerMatchFunc(d.matchVoice, d.handleVoice)
	registrar.RegisterHandlerMatchFunc(d.matchVideoNote, d.handleVideoNote)
	registrar.RegisterHandlerMatchFunc(d.matchAdminCommand, d.handleAdminCommand)
}
