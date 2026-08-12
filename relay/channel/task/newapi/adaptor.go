package newapi

import "github.com/QuantumNous/new-api/relay/channel/task/silkroad"

// TaskAdaptor preserves polling and submission compatibility for historical
// platform-60 SilkRoad tasks without duplicating the provider implementation.
type TaskAdaptor struct {
	silkroad.TaskAdaptor
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}
