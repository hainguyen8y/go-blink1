package blink1

import (
	"testing"
	"time"
)

func TestBlink1(t *testing.T) {
	device, err := OpenNextDevice()
	defer device.Close()

	if err != nil {
		panic(err)
	}

	red := Pattern{
		Red: 20,
		LED: LED1,
		FadeTime: time.Duration(10)*time.Millisecond,
	}

	//device.FadeToRGB(&red)

	blue := Pattern{
		Blue: 20,
		LED: LED2,
		FadeTime: time.Duration(10)*time.Millisecond,
	}
	//device.FadeToRGB(&blue)

	playing, playstart, playend, playcount, playpos, err := device.ReadPlayState()
	if err != nil {
		t.Log(err)
	}
	t.Logf("playing = %d, playstart = %d, playend = %d, playcount = %d, playpos = %d\n",
	playing, playstart, playend, playcount, playpos)

	pats, err := device.ReadPatternAll()
	if err == nil {
		t.Log(pats)
	}

	device.WritePattern(&blue, 3);
	device.WritePattern(&red, 4);

	pats, err = device.ReadPatternAll()
	if err == nil {
		t.Log(pats)
	}

	err = device.Play(1, 3, 4, 0)
	if err != nil {
		t.Log(err)
	}
}
