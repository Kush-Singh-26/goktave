package ui

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	MoveUp    key.Binding
	MoveDown  key.Binding
	Remove    key.Binding
	Clear     key.Binding
	Add       key.Binding
	Play      key.Binding
	Next      key.Binding
	Prev      key.Binding
	Like      key.Binding
	Pause     key.Binding
	Search    key.Binding
	Queue     key.Binding
	Tab1      key.Binding
	Tab2      key.Binding
	Tab3      key.Binding
	Tab4      key.Binding
	Help      key.Binding
	Quit            key.Binding
	CreatePlaylist  key.Binding
	AddToPlaylist   key.Binding
	DeletePlaylist  key.Binding
	PlayPlaylist    key.Binding
	Download        key.Binding
	VolumeUp        key.Binding
	VolumeDown      key.Binding
	SeekForward     key.Binding
	SeekBackward    key.Binding
	Shuffle         key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Search, k.Pause, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Play, k.Prev, k.Next, k.SeekBackward, k.SeekForward, k.Pause, k.Like},
		{k.Search, k.Queue, k.Tab1, k.Tab2, k.Tab3, k.Tab4},
		{k.Add, k.MoveUp, k.MoveDown, k.Remove, k.Clear, k.Shuffle, k.Download},
		{k.CreatePlaylist, k.AddToPlaylist, k.DeletePlaylist, k.PlayPlaylist, k.VolumeDown, k.VolumeUp, k.Help, k.Quit},
	}
}

var Keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	MoveUp: key.NewBinding(
		key.WithKeys("K"),
		key.WithHelp("K", "move up in queue"),
	),
	MoveDown: key.NewBinding(
		key.WithKeys("J"),
		key.WithHelp("J", "move down in queue"),
	),
	Remove: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "remove from queue"),
	),
	Clear: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear queue"),
	),
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add to queue"),
	),
	Play: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "play selected"),
	),
	Next: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next track"),
	),
	Prev: key.NewBinding(
		key.WithKeys("z"),
		key.WithHelp("z", "prev track"),
	),
	Like: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "toggle like"),
	),
	Pause: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "play/pause"),
	),
	Search: key.NewBinding(
		key.WithKeys("s", "/"),
		key.WithHelp("s", "search"),
	),
	Queue: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "focus queue"),
	),
	Tab1: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "results tab"),
	),
	Tab2: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "lyrics tab"),
	),
	Tab3: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "library tab"),
	),
	Tab4: key.NewBinding(
		key.WithKeys("4"),
		key.WithHelp("4", "settings tab"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	CreatePlaylist: key.NewBinding(
		key.WithKeys("C"),
		key.WithHelp("C", "create playlist"),
	),
	AddToPlaylist: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "add to playlist"),
	),
	DeletePlaylist: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "delete playlist"),
	),
	PlayPlaylist: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "play entire playlist"),
	),
	Download: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "download track"),
	),
	VolumeUp: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "volume up"),
	),
	VolumeDown: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "volume down"),
	),
	SeekForward: key.NewBinding(
		key.WithKeys("right", "."),
		key.WithHelp("→/.", "seek forward"),
	),
	SeekBackward: key.NewBinding(
		key.WithKeys("left", ","),
		key.WithHelp("←/,", "seek backward"),
	),
	Shuffle: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "shuffle queue"),
	),
}
