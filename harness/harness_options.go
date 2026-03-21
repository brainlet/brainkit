package harness

type sendOptions struct {
	files          []FileAttachment
	requestContext map[string]any
}

// SendOption configures a SendMessage/Steer/FollowUp call.
type SendOption func(*sendOptions)

// WithFiles attaches files to the message.
func WithFiles(files []FileAttachment) SendOption {
	return func(o *sendOptions) { o.files = files }
}

// WithRequestContext adds request context to the message.
func WithRequestContext(ctx map[string]any) SendOption {
	return func(o *sendOptions) { o.requestContext = ctx }
}

type threadOptions struct {
	title      string
	resourceID string
}

// ThreadOption configures a CreateThread call.
type ThreadOption func(*threadOptions)

// WithThreadTitle sets the thread title.
func WithThreadTitle(title string) ThreadOption {
	return func(o *threadOptions) { o.title = title }
}

// WithThreadResourceID sets the resource ID for the new thread.
func WithThreadResourceID(id string) ThreadOption {
	return func(o *threadOptions) { o.resourceID = id }
}

type listThreadsOptions struct{ resourceID string }

// ListThreadsOption configures a ListThreads call.
type ListThreadsOption func(*listThreadsOptions)

// ForResource filters threads by resource ID.
func ForResource(resourceID string) ListThreadsOption {
	return func(o *listThreadsOptions) { o.resourceID = resourceID }
}

type cloneOptions struct {
	sourceThreadID string
	title          string
	resourceID     string
}

// CloneOption configures a CloneThread call.
type CloneOption func(*cloneOptions)

// CloneFrom specifies the source thread to clone.
func CloneFrom(id string) CloneOption {
	return func(o *cloneOptions) { o.sourceThreadID = id }
}

// CloneWithTitle sets the title for the cloned thread.
func CloneWithTitle(title string) CloneOption {
	return func(o *cloneOptions) { o.title = title }
}

// CloneForResource sets the resource ID for the cloned thread.
func CloneForResource(id string) CloneOption {
	return func(o *cloneOptions) { o.resourceID = id }
}

type listMessagesOptions struct {
	threadID string
	limit    int
}

// ListMessagesOption configures a ListMessages call.
type ListMessagesOption func(*listMessagesOptions)

// ForThread filters messages by thread ID.
func ForThread(id string) ListMessagesOption {
	return func(o *listMessagesOptions) { o.threadID = id }
}

// WithMessageLimit limits the number of messages returned.
func WithMessageLimit(n int) ListMessagesOption {
	return func(o *listMessagesOptions) { o.limit = n }
}

type modelOptions struct {
	scope  string
	modeID string
}

// ModelOption configures a SwitchModel call.
type ModelOption func(*modelOptions)

// ModelScope sets the scope for the model switch ("global", "mode", "thread").
func ModelScope(scope string) ModelOption {
	return func(o *modelOptions) { o.scope = scope }
}

// ModelForMode targets the model switch to a specific mode.
func ModelForMode(modeID string) ModelOption {
	return func(o *modelOptions) { o.modeID = modeID }
}
