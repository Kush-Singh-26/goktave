package mpris

import (
	"fmt"
	"strings"

	"github.com/Kush-Singh-26/goktave/internal/logger"
	"github.com/Kush-Singh-26/goktave/internal/provider"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const intro = `<node>
	<interface name="org.mpris.MediaPlayer2">
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
		<property name="CanGoNext" type="b" access="read"/>
		<property name="CanGoPrevious" type="b" access="read"/>
		<property name="CanPlay" type="b" access="read"/>
		<property name="CanPause" type="b" access="read"/>
		<property name="CanSeek" type="b" access="read"/>
		<property name="CanControl" type="b" access="read"/>
	</interface>
</node>`

type Player struct {
	OnPlayPause func()
	OnNext      func()
	OnPrev      func()
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
			"CanQuit":             {Value: false, Writable: false},
			"CanRaise":            {Value: false, Writable: false},
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
			"CanGoNext":      {Value: true, Writable: false},
			"CanGoPrevious":  {Value: true, Writable: false},
			"CanPlay":        {Value: true, Writable: false},
			"CanPause":       {Value: true, Writable: false},
			"CanSeek":        {Value: false, Writable: false},
			"CanControl":     {Value: true, Writable: false},
		},
	}

	props, err := prop.Export(conn, "/org/mpris/MediaPlayer2", propsSpec)
	if err != nil {
		logger.L.Error("MPRIS Prop Export failed", "err", err)
		return nil, err
	}

	logger.L.Info("MPRIS properties exported successfully")

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

	if err := m.props.Set("org.mpris.MediaPlayer2.Player", "Metadata", dbus.MakeVariant(metadata)); err != nil {
		logger.L.Error("MPRIS Metadata error", "err", err)
	}
}

func (m *Manager) UpdateStatus(status string) {
	if err := m.props.Set("org.mpris.MediaPlayer2.Player", "PlaybackStatus", dbus.MakeVariant(status)); err != nil {
		logger.L.Error("MPRIS Status error", "err", err)
	}
}
