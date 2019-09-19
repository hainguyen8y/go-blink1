package libusb

/*
	#cgo LDFLAGS: -lusb
	#include <usb.h>
	#include <string.h>
*/
import "C"
import (
	"unsafe"
	"fmt"
)

const (
	USBRQ_HID_GET_REPORT		= 0x01
	USBRQ_HID_SET_REPORT        = 0x09
	USB_HID_REPORT_TYPE_FEATURE = 3
	USB_COMM_TIMEOUT_DEFAULT	= 5000
	BLINK1_BUFFER_SIZE 			= 9
)

type Info struct {
	Bus    string
	Device string
	Vid    int
	Pid    int
}

type Device struct {
	*Info
	handle     *C.usb_dev_handle
	descriptor C.struct_usb_device_descriptor
	timeout    int
}

func init() {
	C.usb_init()
}

func RefreshUsbList() {
	C.usb_find_busses()
	C.usb_find_devices()
}

func Enum() []Info {
	fmt.Printf("")

	bus := C.usb_get_busses()
	n := 0
	for ; bus != nil; bus = bus.next {
		for dev := bus.devices; dev != nil; dev = dev.next {
			n += 1
		}
	}
	infos := make([]Info, n)

	bus = C.usb_get_busses()
	n = 0

	for ; bus != nil; bus = bus.next {
		busname := C.GoString(&bus.dirname[0])

		for dev := bus.devices; dev != nil; dev = dev.next {
			devname := C.GoString(&dev.filename[0])

			var info Info
			info.Bus = busname
			info.Device = devname
			info.Vid = int(dev.descriptor.idVendor)
			info.Pid = int(dev.descriptor.idProduct)

			infos[n] = info
			n += 1
		}
	}
	return infos
}

func Open(vid, pid int, device string) *Device {
	for bus := C.usb_get_busses(); bus != nil; bus = bus.next {
		for dev := bus.devices; dev != nil; dev = dev.next {
			if int(dev.descriptor.idVendor) == vid &&
				int(dev.descriptor.idProduct) == pid &&
				C.GoString(&dev.filename[0]) == device {

				h := C.usb_open(dev)
				if h == nil {
					continue;
				}
				drivername := C.malloc(C.sizeof_char*2)
				C.memset(drivername, 0, C.sizeof_char*2)
				C.usb_get_driver_np(h, 0, (*C.char)(drivername), 2)
				len := int(C.strlen((*C.char)(drivername)))
				if len > 0 {
					detachrc := C.usb_detach_kernel_driver_np(h, 0);
					if detachrc != 0 {
						fmt.Printf("detach error %s\n", C.GoString(C.usb_strerror()));
					}
				}
				C.free(drivername)

				rdev := &Device{
					&Info{
						C.GoString(&bus.dirname[0]),
						C.GoString(&dev.filename[0]), vid, pid,
					},
					h, dev.descriptor, USB_COMM_TIMEOUT_DEFAULT,
				}
				return rdev
			}
		}
	}
	return nil
}

func (dev *Device) Close() int {
	r := int(C.usb_close(dev.handle))
	dev.handle = nil
	return r
}

func (dev *Device) String(key int) string {
	buf := make([]C.char, 256)

	C.usb_get_string_simple(
		dev.handle,
		C.int(key),
		&buf[0],
		C.size_t(len(buf)))

	return C.GoString(&buf[0])
}

func (self *Device) Vendor() string {
	return self.String(int(self.descriptor.iManufacturer))
}

func (self *Device) Product() string {
	return self.String(int(self.descriptor.iProduct))
}

func LastError() string {
	return C.GoString(C.usb_strerror())
}

func (self *Device) LastError() string {
	return LastError()
}

func (self *Device) BulkWrite(ep int, dat []byte) int {
	return int(C.usb_bulk_write(self.handle,
		C.int(ep),
		(*C.char)(unsafe.Pointer(&dat[0])),
		C.int(len(dat)),
		C.int(self.timeout)))
}

func (self *Device) BulkRead(ep int, dat []byte) int {
	return int(C.usb_bulk_read(self.handle,
		C.int(ep),
		(*C.char)(unsafe.Pointer(&dat[0])),
		C.int(len(dat)),
		C.int(self.timeout)))
}

func (self *Device) Configuration(conf int) int {
	return int(C.usb_set_configuration(self.handle, C.int(conf)))
}

func (self *Device) Interface(ifc int) int {
	return int(C.usb_claim_interface(self.handle, C.int(ifc)))
}

func (self *Device) ControlMsg(reqtype int, req int, value int, index int, dat []byte) int {
	return int(C.usb_control_msg(self.handle,
		C.int(reqtype),
		C.int(req),
		C.int(value),
		C.int(index),
		(*C.char)(unsafe.Pointer(&dat[0])),
		C.int(len(dat)),
		C.int(self.timeout)))
}

func (self *Device)usbhidSetReport(data []byte) (error) {
	reportID := int(data[0])
	length := C.int(len(data))

	claimrc := C.usb_claim_interface(self.handle, 0);
	if int(claimrc) != 0 {
		return fmt.Errorf("%s", C.GoString(C.usb_strerror()))
	}

	defer C.usb_release_interface(self.handle, 0);

	rc := C.usb_control_msg(self.handle,
		C.int(C.USB_TYPE_CLASS|C.USB_RECIP_INTERFACE|C.USB_ENDPOINT_OUT),
		C.int(USBRQ_HID_SET_REPORT),
		C.int(USB_HID_REPORT_TYPE_FEATURE<<8|(reportID&0xff)),
		C.int(0),
		(*C.char)(unsafe.Pointer(&data[0])),
		length,
		C.int(self.timeout))
	if rc != length {
		return fmt.Errorf("%s", C.GoString(C.usb_strerror()))
	}
	return nil
}

func (self *Device)usbhidGetReport(reportNumber int, len int) ([]byte, error) {
	data := C.malloc(C.sizeof_char * C.ulong(len))
	defer C.free(data)

	claimrc := C.usb_claim_interface(self.handle, 0);
	if int(claimrc) != 0 {
		return nil, fmt.Errorf("%s", C.GoString(C.usb_strerror()))
	}

	defer C.usb_release_interface(self.handle, 0);

	bytesReceived := C.usb_control_msg(self.handle,
		C.int(C.USB_TYPE_CLASS | C.USB_RECIP_INTERFACE | C.USB_ENDPOINT_IN),
		C.int(USBRQ_HID_GET_REPORT),
		C.int(USB_HID_REPORT_TYPE_FEATURE << 8 | (reportNumber & 0xff)),
		C.int(0),
		(*C.char)(data),
		C.int(len),
		C.int(self.timeout))
	if bytesReceived < 0 {
		return nil, fmt.Errorf("%s", C.GoString(C.usb_strerror()))
	}
	return C.GoBytes(data, bytesReceived), nil
}

func (self *Device)Blink1Write(data []byte) (error) {
	return self.usbhidSetReport(data)
}

func (self *Device)Blink1WriteRead(data []byte) ([]byte, error) {
	reportId := int(data[0])
	err := self.usbhidSetReport(data)
	if err != nil {
		return nil, err
	}
	return self.usbhidGetReport(reportId, BLINK1_BUFFER_SIZE)
}
