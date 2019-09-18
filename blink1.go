package blink1

import (
	"errors"
	"time"
	"blink1/libusb"
)

const BLINK1_REPORT_ID = 1

// USB IDs
const (
	USBVendorID  = 10168
	USBProductID = 493
)

var (
	errNoDevices = errors.New("No Blink(1) device found or all already in use")
	openDevices     = make(map[string]*Device)
	defaultFadeTime = 10
)

const (
	LEDAll uint8 = iota
	LED1
	LED2
)

type Pattern struct {
	Red      uint8         // Red value 0-255
	Green    uint8         // Green value 0-255
	Blue     uint8         // Blue value 0-255
	LED      uint8         // which LED to address (0=all, 1=1st LED, 2=2nd LED)
	FadeTime time.Duration // Fadetime to state
	Duration time.Duration // Duration of state after FadeTime
}

// Device Thingm Blink(1) USB device
type Device struct {
	Device          *libusb.Device // USB device
	DefaultFadeTime time.Duration  // Default time to fade between states
	CurrentState    Pattern        // Current state of the Blink(1)
}

// OpenNextDevice opens and returns the next available Blink(1) device
func OpenNextDevice() (device *Device, err error) {
	libusb.RefreshUsbList()
	// Enum devices and look for next Blink(1)
	for _, dev := range libusb.Enum() {
		if dev.Vid == USBVendorID && dev.Pid == USBProductID {
			if openDevices[dev.Device] == nil {
				d := libusb.Open(dev.Vid, dev.Pid, dev.Device)
				if d != nil {
					device = &Device{
						Device:          d,
						DefaultFadeTime: time.Duration(defaultFadeTime) * time.Millisecond,
					}
					openDevices[dev.Device] = device
					return
				}
			}
		}
	}
	err = errNoDevices
	return
}

// Close communication channel to Blink(1)
func (b *Device) Close() {
	delete(openDevices, b.Device.Device)
	_ = b.Device.Close()
}

func (self *Device) SetLed(id int) (error) {
	cmd := []byte{ BLINK1_REPORT_ID, 'l', byte(id),0,0, 0,0,0 };
	err := self.Device.Blink1Write(cmd)
	return err
}

func (self *Device) ReadPlayState() ( playing, playstart, playend, playcount, playpos int, err error) {
	cmd := []byte{ BLINK1_REPORT_ID, 'S', 0,0,0, 0,0,0 };

	buf, err := self.Device.Blink1WriteRead(cmd)
	if err != nil {
		return
	}
	playing	  = int(buf[2])
	playstart = int(buf[3])
	playend   = int(buf[4])
	playcount = int(buf[5])
	playpos   = int(buf[6])
	return
}

func (self *Device) ReadPattern(pos int) (*Pattern, error) {
	cmd := []byte{ BLINK1_REPORT_ID, 'R', 0,0,0, 0,0, byte(pos&0xff)};
	buf, err := self.Device.Blink1WriteRead(cmd)

	if err != nil {
		return nil, err
	}

	pat := &Pattern{}

	pat.Red = uint8(buf[2])
	pat.Green = uint8(buf[3])
	pat.Blue = uint8(buf[4])
	pat.LED = uint8(buf[7])
	pat.FadeTime = time.Millisecond*time.Duration(int(((int(buf[5])<<8) + (int(buf[6]) &0xff)) * 10))
	return pat, nil
}

func (self *Device) ReadPatternAll() ([]Pattern, error) {
	var pats []Pattern
	for i:= 0; i < 32; i++ {
		pat, err := self.ReadPattern(i)
		if err != nil {
			return nil, err
		}
		pats = append(pats, *pat)
	}
	return pats, nil
}

func (self *Device) WritePattern(pat *Pattern, pos int) error {
	err := self.SetLed(int(pat.LED))
	if err != nil {
		return err
	}

	dms := int(pat.FadeTime/(10*time.Millisecond));
	cmd := []byte{ BLINK1_REPORT_ID, 'P',
		pat.Red, pat.Green, pat.Blue,
		byte(dms>>8), byte(dms % 0xff), byte(pos&0xff)};
	return self.Device.Blink1Write(cmd)
}

func (self *Device) WritePatternAll(pats []Pattern) (error) {
	for i, pat := range pats {
		err := self.WritePattern(&pat, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *Device) Play(play, startpos, endpos, count uint8) error {
	cmd := []byte{ BLINK1_REPORT_ID, 'p', byte(play), byte(startpos), byte(endpos), byte(count),0, 0};
	return self.Device.Blink1Write(cmd)
}

func (self *Device) FadeToRGB(pat *Pattern) error {
	dms := pat.FadeTime / 10
	cmd := []byte{
		1, 'c', byte(pat.Red), byte(pat.Green), byte(pat.Blue), byte(dms >> 8), byte(dms % 127), byte(pat.LED),
	}
	err := self.Device.Blink1Write(cmd)
	return err
}
