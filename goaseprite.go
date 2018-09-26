package goaseprite

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	// PlayForward plays animations forward
	PlayForward = "forward"
	// PlayBackward plays animations backwards
	PlayBackward = "reverse"
	// PlayPingPong plays animation forward then backward
	PlayPingPong = "pingpong"
)

// Frame contains timing and position information for the frame on the spritesheet.
type Frame struct {
	X        int32
	Y        int32
	Duration float32
}

// Animation contains details regarding each animation from Aseprite. This represents, essentially, a tag in Aseprite.
type Animation struct {
	Name      string
	Start     int32
	End       int32
	Direction string
}

// File contains all properties of an exported aseprite file.
type File struct {
	ImagePath        string
	FrameWidth       int32
	FrameHeight      int32
	Frames           []Frame
	Animations       []Animation
	CurrentAnimation *Animation
	frameCounter     float32
	CurrentFrame     int32
	prevCurrentFrame int32
	PrevFrame        int32
	PlaySpeed        float32
	Playing          bool
	pingpongedOnce   bool
}

// Play queues playback of the specified animation (assuming it's in the File).
func (asf *File) Play(animName string) {
	cur := asf.GetAnimation(animName)
	if cur == nil {
		log.Fatal(`Error: Animation named "` + animName + `" not found in Aseprite file!`)
	}
	if asf.CurrentAnimation != cur {
		asf.CurrentAnimation = cur
		asf.CurrentFrame = asf.CurrentAnimation.Start
		if asf.CurrentAnimation.Direction == PlayBackward {
			asf.CurrentFrame = asf.CurrentAnimation.End
		}
		asf.pingpongedOnce = false
	}
}

// Update steps the file forward in time, updating the currently playing
// animation (and also handling looping).
func (asf *File) Update(deltaTime float32) {

	asf.PrevFrame = asf.prevCurrentFrame
	asf.prevCurrentFrame = asf.CurrentFrame

	if asf.CurrentAnimation != nil {

		asf.frameCounter += deltaTime * asf.PlaySpeed

		anim := asf.CurrentAnimation

		if asf.frameCounter > asf.Frames[asf.CurrentFrame].Duration {

			asf.frameCounter = 0

			if anim.Direction == PlayForward {
				asf.CurrentFrame++
			} else if anim.Direction == PlayBackward {
				asf.CurrentFrame--
			} else if anim.Direction == PlayPingPong {
				if asf.pingpongedOnce {
					asf.CurrentFrame--
				} else {
					asf.CurrentFrame++
				}
			}

		}

		if asf.CurrentFrame > anim.End {
			if anim.Direction == PlayPingPong {
				asf.pingpongedOnce = !asf.pingpongedOnce
				asf.CurrentFrame = anim.End - 1
				if asf.CurrentFrame < anim.Start {
					asf.CurrentFrame = anim.Start
				}
			} else {
				asf.CurrentFrame = anim.Start
			}
		}

		if asf.CurrentFrame < anim.Start {
			if anim.Direction == PlayPingPong {
				asf.pingpongedOnce = !asf.pingpongedOnce
				asf.CurrentFrame = anim.Start + 1
				if asf.CurrentFrame > anim.End {
					asf.CurrentFrame = anim.End
				}

			} else {
				asf.CurrentFrame = anim.End
			}
		}

	}

}

// GetAnimation returns a pointer to an Animation of the desired name. If it can't
// be found, it will return `nil`.
func (asf *File) GetAnimation(animName string) *Animation {

	for index := range asf.Animations {
		anim := &asf.Animations[index]
		if anim.Name == animName {
			return anim
		}
	}

	return nil

}

// GetFrameXY returns the current frame's X and Y coordinates on the source sprite sheet
// for drawing the sprite.
func (asf *File) GetFrameXY() (int32, int32) {

	var frameX, frameY int32 = -1, -1

	if asf.CurrentAnimation != nil {

		frameX = asf.Frames[asf.CurrentFrame].X
		frameY = asf.Frames[asf.CurrentFrame].Y

	}

	return frameX, frameY

}

// IsPlaying returns if the named animation is playing.
func (asf *File) IsPlaying(animName string) bool {

	if asf.CurrentAnimation != nil && asf.CurrentAnimation.Name == animName {
		return true
	}

	return false
}

// TouchingTag returns if the File's playback is touching a tag by the specified name.
func (asf *File) TouchingTag(tagName string) bool {
	for _, anim := range asf.Animations {
		if anim.Name == tagName && asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End {
			return true
		}
	}
	return false
}

// TouchingTags returns a list of tags the playback is touching.
func (asf *File) TouchingTags() []string {
	anims := []string{}
	for _, anim := range asf.Animations {
		if asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End {
			anims = append(anims, anim.Name)
		}
	}
	return anims
}

// HitTag returns if the File's playback just touched a tag by the specified name.
func (asf *File) HitTag(tagName string) bool {
	for _, anim := range asf.Animations {
		if anim.Name == tagName && (asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End) && (asf.PrevFrame < anim.Start || asf.PrevFrame > anim.End) {
			return true
		}
	}
	return false
}

// HitTags returns a list of tags the File just touched.
func (asf *File) HitTags() []string {
	anims := []string{}

	for _, anim := range asf.Animations {
		if (asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End) && (asf.PrevFrame < anim.Start || asf.PrevFrame > anim.End) {
			anims = append(anims, anim.Name)
		}
	}
	return anims
}

// LeftTag returns if the File's playback just left a tag by the specified name.
func (asf *File) LeftTag(tagName string) bool {
	for _, anim := range asf.Animations {
		if anim.Name == tagName && (asf.PrevFrame >= anim.Start && asf.PrevFrame <= anim.End) && (asf.CurrentFrame < anim.Start || asf.CurrentFrame > anim.End) {
			return true
		}
	}
	return false
}

// LeftTags returns a list of tags the File just left.
func (asf *File) LeftTags() []string {
	anims := []string{}

	for _, anim := range asf.Animations {
		if (asf.PrevFrame >= anim.Start && asf.PrevFrame <= anim.End) && (asf.CurrentFrame < anim.Start || asf.CurrentFrame > anim.End) {
			anims = append(anims, anim.Name)
		}
	}
	return anims
}

func readFile(filePath string) string {

	file, err := os.Open(filePath)

	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)

	out := ""

	for scanner.Scan() {
		out += scanner.Text()
	}

	file.Close()

	return out

}

type byFrameNumber []string

func (b byFrameNumber) Len() int {
	return len(b)
}
func (b byFrameNumber) Swap(x, y int) {
	b[x], b[y] = b[y], b[x]
}
func (b byFrameNumber) Less(xi, yi int) bool {
	x := b[xi]
	y := b[yi]
	xfi := strings.LastIndex(x, " ") + 1
	xli := strings.LastIndex(x, ".")
	xv, _ := strconv.ParseInt(x[xfi:xli], 10, 32)
	yfi := strings.LastIndex(y, " ") + 1
	yli := strings.LastIndex(y, ".")
	yv, _ := strconv.ParseInt(y[yfi:yli], 10, 32)
	return xv < yv
}

// Load parses and returns an File for a supplied JSON exported from Aseprite. This is your starting point.
// goaseprite is set up to read JSONs for sprite sheets exported with the Hash type.
func Load(aseJSONFilePath string) File {

	file := readFile(aseJSONFilePath)

	ase := File{}
	ase.Animations = make([]Animation, 0)
	ase.PlaySpeed = 1

	if path, err := filepath.Abs(gjson.Get(file, "meta.image").String()); err != nil {
		log.Fatalln(err)
	} else {
		ase.ImagePath = path
	}

	frameNames := []string{}

	for key := range gjson.Get(file, "frames").Map() {
		frameNames = append(frameNames, key)
	}

	sort.Sort(byFrameNumber(frameNames))

	for _, key := range frameNames {

		frameName := key
		frameName = strings.Replace(frameName, ".", `\.`, -1)
		frameData := gjson.Get(file, "frames."+frameName)

		frame := Frame{}
		frame.X = int32(frameData.Get("frame.x").Num)
		frame.Y = int32(frameData.Get("frame.y").Num)
		frame.Duration = float32(frameData.Get("duration").Num) / 1000

		ase.Frames = append(ase.Frames, frame)

		if ase.FrameWidth == 0 {
			ase.FrameWidth = int32(frameData.Get("sourceSize.w").Num)
			ase.FrameHeight = int32(frameData.Get("sourceSize.h").Num)
		}

	}

	for _, anim := range gjson.Get(file, "meta.frameTags").Array() {

		ase.Animations = append(ase.Animations, Animation{
			Name:      anim.Get("name").Str,
			Start:     int32(anim.Get("from").Num),
			End:       int32(anim.Get("to").Num),
			Direction: anim.Get("direction").Str})

	}

	return ase
}
