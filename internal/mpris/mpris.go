package mpris

import (
	"fmt"
	"strings"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/logger"
	"github.com/Kush-Singh-26/goktave/internal/provider"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const intro = `<node>
	<interface name="org.mpris.MediaPlayer2">
		<method name="Raise"></method>
		<method name="Quit"></method>
		<property name="CanQuit" type="b" access="read"/>
		<property name="CanRaise" type="b" access="read"/>
		<property name="HasTrackList" type="b" access="read"/>
		<property name="Identity" type="s" access="read"/>
		<property name="DesktopEntry" type="s" access="read"/>
		<property name="SupportedUriSchemes" type="as" access="read"/>
		<property name="SupportedMimeTypes" type="as" access="read"/>
	</interface>
	<interface name="org.mpris.MediaPlayer2.Player">
		<method name="PlayPause"></method>
		<method name="Next"></method>
		<method name="Previous"></method>
		<method name="Stop"></method>
		<method name="Seek">
			<arg direction="in" name="Offset" type="x"/>
		</method>
		<method name="SetPosition">
			<arg direction="in" name="TrackId" type="o"/>
			<arg direction="in" name="Position" type="x"/>
		</method>
		<method name="OpenUri">
			<arg direction="in" name="Uri" type="s"/>
		</method>
		<property name="PlaybackStatus" type="s" access="read"/>
		<property name="Metadata" type="a{sv}" access="read"/>
		<property name="Volume" type="d" access="readwrite"/>
		<property name="Position" type="x" access="read"/>
		<property name="CanGoNext" type="b" access="read"/>
		<property name="CanGoPrevious" type="b" access="read"/>
		<property name="CanPlay" type="b" access="read"/>
		<property name="CanPause" type="b" access="read"/>
		<property name="CanSeek" type="b" access="read"/>
		<property name="CanControl" type="b" access="read"/>
	</interface>
</node>`

type Root struct {
	OnRaise func()
	OnQuit  func()
}

func (r *Root) Raise() *dbus.Error {
	if r.OnRaise != nil {
		r.OnRaise()
	}
	return nil
}

func (r *Root) Quit() *dbus.Error {
	if r.OnQuit != nil {
		r.OnQuit()
	}
	return nil
}

type Player struct {
	OnPlayPause func()
	OnNext      func()
	OnPrev      func()
	OnRaise     func()
	OnQuit      func()
}

func (p *Player) PlayPause() *dbus.Error {
	if p.OnPlayPause != nil {
		p.OnPlayPause()
	}
	return nil
}

func (p *Player) Next() *dbus.Error {
	if p.OnNext != nil {
		p.OnNext()
	}
	return nil
}

func (p *Player) Previous() *dbus.Error {
	if p.OnPrev != nil {
		p.OnPrev()
	}
	return nil
}
func (p *Player) Stop()     *dbus.Error                     { return nil }
func (p *Player) Seek(offset int64) *dbus.Error             { return nil }
func (p *Player) SetPosition(id dbus.ObjectPath, pos int64) *dbus.Error { return nil }
func (p *Player) OpenUri(uri string) *dbus.Error            { return nil }

type Manager struct {
	conn   *dbus.Conn
	player *Player
	props  *prop.Properties
}

func Start(onPlayPause func(), onNext func(), onPrev func()) (*Manager, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}

	reply, err := conn.RequestName("org.mpris.MediaPlayer2.goktave", dbus.NameFlagReplaceExisting|dbus.NameFlagAllowReplacement)
	if err != nil {
		return nil, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner && reply != dbus.RequestNameReplyAlreadyOwner {
		return nil, fmt.Errorf("could not take bus name: reply %v", reply)
	}

	player := &Player{OnPlayPause: onPlayPause, OnNext: onNext, OnPrev: onPrev}

	propsSpec := map[string]map[string]*prop.Prop{
		"org.mpris.MediaPlayer2": {
			"CanQuit":             {Value: true, Writable: false},
			"CanRaise":            {Value: true, Writable: false},
			"HasTrackList":        {Value: false, Writable: false},
			"Identity":            {Value: "GoKtave", Writable: false},
			"DesktopEntry":        {Value: "goktave", Writable: false},
			"SupportedUriSchemes": {Value: []string{"https"}, Writable: false},
			"SupportedMimeTypes":  {Value: []string{"audio/mpeg", "audio/ogg", "audio/webm"}, Writable: false},
		},
		"org.mpris.MediaPlayer2.Player": {
			"PlaybackStatus": {Value: "Stopped", Writable: true, Emit: prop.EmitTrue},
			"Metadata":       {Value: map[string]dbus.Variant{}, Writable: true, Emit: prop.EmitTrue},
			"Volume":         {Value: 1.0, Writable: true},
			"Position":       {Value: int64(0), Writable: false},
			"CanGoNext":      {Value: true, Writable: false},
			"CanGoPrevious":  {Value: true, Writable: false},
			"CanPlay":        {Value: true, Writable: false},
			"CanPause":       {Value: true, Writable: false},
			"CanSeek":        {Value: true, Writable: false},
			"CanControl":     {Value: true, Writable: false},
		},
	}

	props, err := prop.Export(conn, "/org/mpris/MediaPlayer2", propsSpec)
	if err != nil {
		logger.L.Error("MPRIS Prop Export failed", "err", err)
		return nil, err
	}

	logger.L.Info("MPRIS properties exported successfully")

	root := &Root{}
	conn.Export(root, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2")
	conn.Export(player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")
	conn.Export(introspect.Introspectable(intro), "/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Introspectable")

	return &Manager{conn: conn, player: player, props: props}, nil
}

func (m *Manager) UpdateMetadata(track *provider.Track) {
	if track == nil {
		return
	}

	logger.L.Debug("Updating MPRIS metadata", "title", track.Title)

	// D-Bus object paths only allow [A-Z][a-z][0-9]_ and components cannot start with a digit
	safeID := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, track.VideoID)

	metadata := map[string]dbus.Variant{
		"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/org/goktave/track/t" + safeID)),
		"mpris:length":  dbus.MakeVariant(int64(track.Duration) * 1000000),
		"xesam:title":   dbus.MakeVariant(track.Title),
		"xesam:album":   dbus.MakeVariant(track.Album),
		"xesam:artist":  dbus.MakeVariant([]string{track.Artist}),
		"mpris:artUrl":  dbus.MakeVariant(fmt.Sprintf("https://img.youtube.com/vi/%s/0.jpg", track.VideoID)),
		"xesam:url":     dbus.MakeVariant(fmt.Sprintf("https://www.youtube.com/watch?v=%s", track.VideoID)),
	}

	// SetMust bypasses the Writable check (which only applies to external D-Bus Set calls).
	m.props.SetMust("org.mpris.MediaPlayer2.Player", "Metadata", metadata)
}

func (m *Manager) UpdateStatus(status string) {
	// SetMust is the internal Go setter — it bypasses Writable checks and
	// emits PropertiesChanged if the property has Emit: EmitTrue.
	m.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", status)
}

func (m *Manager) UpdatePosition(pos time.Duration) {
	// Position is Writable:false (D-Bus clients cannot set it), so we MUST use
	// SetMust instead of Set. Using Set would return ErrReadOnly and silently
	// leave Position stuck at 0 in the D-Bus property cache forever.
	m.props.SetMust("org.mpris.MediaPlayer2.Player", "Position", pos.Microseconds())
}

// EmitSeeked broadcasts the org.mpris.MediaPlayer2.Player.Seeked signal.
// This tells media widgets (playerctl, Niri media controls, etc.) the exact
// current position so they do not extrapolate from a stale cached value.
// Call this whenever playback resumes from pause.
func (m *Manager) EmitSeeked(pos time.Duration) {
	usec := pos.Microseconds()
	// Update the cached property so Get("Position") agrees.
	m.props.SetMust("org.mpris.MediaPlayer2.Player", "Position", usec)
	// Emit the Seeked signal on the Player object path.
	err := m.conn.Emit(
		"/org/mpris/MediaPlayer2",
		"org.mpris.MediaPlayer2.Player.Seeked",
		usec,
	)
	if err != nil {
		logger.L.Error("MPRIS Seeked signal error", "err", err)
	}
}
