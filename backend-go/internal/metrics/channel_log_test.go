package metrics

import "testing"

func TestChannelLogStoreRemoveAndShift(t *testing.T) {
	store := NewChannelLogStore()
	store.Record(0, &ChannelLog{Model: "channel-0"})
	store.Record(1, &ChannelLog{Model: "channel-1"})
	store.Record(2, &ChannelLog{Model: "channel-2"})

	store.RemoveAndShift(1)

	channel0Logs := store.Get(0)
	if len(channel0Logs) != 1 || channel0Logs[0].Model != "channel-0" {
		t.Fatalf("channel 0 logs = %#v, want channel-0", channel0Logs)
	}

	shiftedLogs := store.Get(1)
	if len(shiftedLogs) != 1 || shiftedLogs[0].Model != "channel-2" {
		t.Fatalf("shifted logs = %#v, want channel-2", shiftedLogs)
	}

	if got := store.Get(2); got != nil {
		t.Fatalf("channel 2 logs = %#v, want nil", got)
	}
}
