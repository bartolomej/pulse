package widgets

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"math"
	"regexp"
	"strconv"
)

var hslColorPattern = regexp.MustCompile(`^(?:hsla?\()?([\d\.]+)(?: |,)+([\d\.]+)%?(?: |,)+([\d\.]+)%?\)?$`)

const (
	hslHueMax        = 360
	hslSaturationMax = 100
	hslLightnessMax  = 100
)

type HSLColor struct {
	H float64
	S float64
	L float64
}

func (c *HSLColor) String() string {
	return fmt.Sprintf("hsl(%.1f, %.1f%%, %.1f%%)", c.H, c.S, c.L)
}

func (c *HSLColor) ToHex() string {
	return hslToHex(c.H, c.S, c.L)
}

func (c1 *HSLColor) SameAs(c2 *HSLColor) bool {
	if c1 == nil && c2 == nil {
		return true
	}
	if c1 == nil || c2 == nil {
		return false
	}
	return c1.H == c2.H && c1.S == c2.S && c1.L == c2.L
}

func (c *HSLColor) UnmarshalYAML(node *yaml.Node) error {
	var value string

	if err := node.Decode(&value); err != nil {
		return err
	}

	matches := hslColorPattern.FindStringSubmatch(value)

	if len(matches) != 4 {
		return fmt.Errorf("invalid HSL color format: %s", value)
	}

	hue, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return err
	}

	if hue > hslHueMax {
		return fmt.Errorf("HSL hue must be between 0 and %d", hslHueMax)
	}

	saturation, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return err
	}

	if saturation > hslSaturationMax {
		return fmt.Errorf("HSL saturation must be between 0 and %d", hslSaturationMax)
	}

	lightness, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return err
	}

	if lightness > hslLightnessMax {
		return fmt.Errorf("HSL lightness must be between 0 and %d", hslLightnessMax)
	}

	c.H = hue
	c.S = saturation
	c.L = lightness

	return nil
}

func hslToHex(h, s, l float64) string {
	s /= 100.0
	l /= 100.0

	var r, g, b float64

	if s == 0 {
		r, g, b = l, l, l
	} else {
		hueToRgb := func(p, q, t float64) float64 {
			if t < 0 {
				t += 1
			}
			if t > 1 {
				t -= 1
			}
			if t < 1.0/6.0 {
				return p + (q-p)*6.0*t
			}
			if t < 1.0/2.0 {
				return q
			}
			if t < 2.0/3.0 {
				return p + (q-p)*(2.0/3.0-t)*6.0
			}
			return p
		}

		q := 0.0
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}

		p := 2*l - q

		h /= 360.0

		r = hueToRgb(p, q, h+1.0/3.0)
		g = hueToRgb(p, q, h)
		b = hueToRgb(p, q, h-1.0/3.0)
	}

	ir := int(math.Round(r * 255.0))
	ig := int(math.Round(g * 255.0))
	ib := int(math.Round(b * 255.0))

	ir = int(math.Max(0, math.Min(255, float64(ir))))
	ig = int(math.Max(0, math.Min(255, float64(ig))))
	ib = int(math.Max(0, math.Min(255, float64(ib))))

	return fmt.Sprintf("#%02x%02x%02x", ir, ig, ib)
}
