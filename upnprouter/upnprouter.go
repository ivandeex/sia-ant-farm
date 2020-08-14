package upnprouter

import (
	"log"
	"net"

	"gitlab.com/NebulousLabs/errors"
	"gitlab.com/NebulousLabs/go-upnp"
)

var (
	// UPnPEnabled is a flag to store whether we have UPnP enabled router to
	// save UPnP operations when the router is not enabled
	UPnPEnabled = true
)

// CheckUPnPEnabled checks wheteher there is UPnP enabled router connected and
// sets the flag accordingly
func CheckUPnPEnabled() {
	_, err := upnp.Discover()
	if err != nil {
		UPnPEnabled = false
		log.Printf("[INFO] [ant-farm] UPnP enabled router is not available: %v", err)
	} else {
		log.Println("[INFO] [ant-farm] UPnP enabled router is available")
	}
}

// ClearPorts clears ports on UPnP enabled router
func ClearPorts(addresses ...*net.TCPAddr) error {
	upnprouter, err := upnp.Discover()
	if err != nil {
		return errors.AddContext(err, "can't discover UPnP enabled router")
	}
	for _, a := range addresses {
		err = upnprouter.Clear(uint16(a.Port))
		if err != nil {
			return err
		}
	}
	return nil
}
