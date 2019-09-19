package blink1

import (
	"testing"
	"time"
)

func TestBlink1(t *testing.T) {
	device, err := OpenNextDevice()
	defer device.Close()

	if err != nil {
		t.Fatal(err)
	}

	red := Pattern{
		Red: 20,
		LED: LED1,
		FadeTime: time.Duration(500)*time.Millisecond,
	}

	blue := Pattern{
		Blue: 20,
		LED: LED2,
		FadeTime: time.Duration(500)*time.Millisecond,
	}

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

	device.WritePattern(&blue, 3)
	device.WritePattern(&red, 4)
	device.WritePattern(&Pattern{
		FadeTime: time.Duration(1000)*time.Millisecond,
	}, 5)

	pats, err = device.ReadPatternAll()
	if err == nil {
		t.Log(pats)
	}

	err = device.Play(1, 3, 5, 0)
	if err != nil {
		t.Log(err)
	}
}
