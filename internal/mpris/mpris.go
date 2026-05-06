package mpris

import (
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

// This XML tells Linux what buttons our app supports
const intro = `<node>
	<interface name="org.mpris.MediaPlayer2">
		<property name="CanQuit" type="b" access="read"/>
		<property name="CanRaise" type="b" access="read"/>
	</interface>
	<interface name="org.mpris.MediaPlayer2.Player">
		<method name="PlayPause"></method>
		<method name="Next"></method>
		<property name="CanGoNext" type="b" access="read"/>
		<property name="CanPlay" type="b" access="read"/>
		<property name="CanPause" type="b" access="read"/>
	</interface>
</node>`

type Player struct {
	OnPlayPause func()
	OnNext      func()
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

// Start opens a connection to the Linux desktop bus
func Start(onPlayPause func(), onNext func()) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}

	// Register our app name on the system
	reply, err := conn.RequestName("org.mpris.MediaPlayer2.goktave", dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		return nil // D-Bus is likely unavailable or name taken, fail silently
	}

	player := &Player{OnPlayPause: onPlayPause, OnNext: onNext}
	
	// Export our player to the system
	conn.Export(player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")
	conn.Export(introspect.Introspectable(intro), "/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Introspectable")
	
	return nil
}