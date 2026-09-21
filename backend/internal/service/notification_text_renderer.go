// notification_text_renderer.go — The sending side's view of the notification
// template store.
package service

// notificationTextRenderer is the narrow port every notification sender depends
// on. Senders only render; they never list or update templates, so they are
// wired against this instead of the whole NotificationTemplateService.
//
// Implementations must not fail: a template that cannot be resolved degrades to
// the catalog default rather than blocking the notification.
type notificationTextRenderer interface {
	Render(key string, vars map[string]string) RenderedTemplate
}
