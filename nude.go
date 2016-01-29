package gonude

import (
	"github.com/nfnt/resize"
	"image"
	"math"
	"sort"
    "log"
)

type Region struct {
	Id    uint16
	Count int
}

type Regions []*Region

type SkinImg struct {
	Img         *image.Image
	SkinRegions Regions
}

func (r Regions) Len() int {
	return len(r)
}

func (r Regions) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Regions) Less(i, j int) bool {
	return r[i].Count < r[j].Count
}

func normalizedRgb(r, g, b uint32) (nr, ng, nb float64) {
	fr := float64(r)
	fg := float64(g)
	fb := float64(b)

	if fr == 0 {
		fr = 0.0001
	}
	if fg == 0 {
		fg = 0.0001
	}
	if fb == 0 {
		fb = 0.0001
	}
	sum := fr + fb + fg
	nr = fr / sum
	ng = fg / sum
	nb = fb / sum

	return
}

func toHSV(r, g, b uint32) (h, s, v float64) {
	fr := float64(r)
	fg := float64(g)
	fb := float64(b)
	//hue
	h = math.Acos((0.5 * ((fr - fg) + (fr - fb))) / (math.Sqrt((math.Pow((fr-fg), 2) + ((fr - fb) * (fg - fb))))))

	// saturation
	s = 1 - (3 * ((min3(r, g, b)) / (fr + fg + fb)))
	// value
	v = (1 / 3) * (fr + fg + fb)

	h = 0
	_sum := fr + fg + fb
	_max := max3(r, g, b)
	_min := min3(r, g, b)
	diff := _max - _min
	if _sum == 0 {
		_sum = 0.0001
	}

	if _max == fr {
		if diff == 0 {
			h = math.MaxFloat64
		} else {
			h = (fg - fb) / diff
		}
	} else if _max == fg {
		h = 2 + ((fg - fr) / diff)
	} else {
		h = 4 + ((fr - fg) / diff)
	}

	h *= 60
	if h < 0 {
		h += 360
	}
	s = 1.0 - (3.0 * (_min / _sum))
	v = (1.0 / 3.0) * _max
	return
}

func toYCbCr(r, g, b uint32) (y, cb, cr float64) {
	y = 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	cb = 128 - 0.168736*float64(r) - 0.331364*float64(g) + 0.5*float64(b)
	cr = 128 + 0.5*float64(r) - 0.418688*float64(g) - 0.081312*float64(b)
	return y, cb, cr
}

func max3(a, b, c uint32) float64 {
	fa := float64(a)
	fb := float64(b)
	fc := float64(c)
	return math.Max(math.Max(fa, fb), fc)
}

func min3(a, b, c uint32) float64 {
	fa := float64(a)
	fb := float64(b)
	fc := float64(c)
	return math.Min(math.Min(fa, fb), fc)
}

func classifySkin(r, g, b uint32) bool {
	rgb_classifier := r > 95 && g > 40 && b > 20 && max3(r, g, b)-min3(r, g, b) > 15 && math.Abs(float64(r-g)) > 15 && r > g && r > b

	rgb_classifier2 := r > 220 && g > 210 && b > 170 && math.Abs(float64(r-g)) <= 15 && r > b && g > b

	nr, ng, _ := normalizedRgb(r, g, b)
	normRgbClassifier := (((nr / ng) > 1.185) && ((float64(r*b) / (math.Pow(float64(r+g+b), 2))) > 0.107) && ((float64(r*g) / (math.Pow(float64(r+g+b), 2))) > 0.112))

	h, s, _ := toHSV(r, g, b)
	hsv_classifier := h > 0 && h < 35 && s > 0.23 && s < 0.68

	_, cb, cr := toYCbCr(r, g, b)
	ycbcr_classifier := 97.5 <= cb && cb <= 142.5 && 134 <= cr && cr <= 176

	return rgb_classifier || rgb_classifier2 || normRgbClassifier || hsv_classifier || ycbcr_classifier
}

func (ski *SkinImg) scanImage() bool {
	img := resize.Thumbnail(800, 800, *ski.Img, resize.NearestNeighbor)
	bounds := img.Bounds()
	width := bounds.Size().X
	height := bounds.Size().Y
	totalPixels := width * height

	regionMap := make([]uint16, totalPixels)
	linked := make([]uint16, 1)

	var currentLabel uint16 = 1

	//Label components
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			index := (y-bounds.Min.Y)*width + (x - bounds.Min.X)
			var checkIndex [4]int

			//init checkedIndex
			for i := range checkIndex {
				nx := (i%3 - 1) + x
				ny := y + i/3 - 1

				if nx < bounds.Min.X || ny < bounds.Min.Y || nx >= bounds.Max.X || ny >= bounds.Max.Y {
					checkIndex[i] = -1
				} else {
					checkIndex[i] = (ny-bounds.Min.Y)*width + (nx - bounds.Min.X)
				}
			}

			if classifySkin(r, g, b) {
				min := uint16(math.MaxInt16)
				l := make([]uint16, 0)
				for _, cindex := range checkIndex {
					if cindex != -1 {
						val := regionMap[cindex]
						if val != 0 {
							found := false
							for _, v := range l {
								if val == v {
									found = true
									break
								}
							}

							if !found {
								l = append(l, val)
							}

							if val < min {
								min = val
							}
						}
					}
				}

				if min != uint16(math.MaxInt16) {
					regionMap[index] = min
					for _, v := range l {
						linked[v] = linked[min]
					}
				} else {
					regionMap[index] = currentLabel
					linked = append(linked, currentLabel)
					currentLabel++
				}
			}
		}
	}

	//Merge
	var skinRegions Regions
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			index := (y-bounds.Min.Y)*width + (x - bounds.Min.X)
			if regionMap[index] != 0 {
				regionMap[index] = linked[regionMap[index]]

				//search region
				found := false
				for _, r := range skinRegions {
					if r.Id == regionMap[index] {
						r.Count++

						found = true
						break
					}
				}

				if !found {
					skinRegions = append(skinRegions, &Region{regionMap[index], 1})
				}
			}
		}
	}
	//log.Println("component merged")

	//reduce noise
	for _, region := range skinRegions {
		if region.Count > 9 { //at least 3x3
			ski.SkinRegions = append(ski.SkinRegions, region)
		}
	}

	return ski.analyseRegions()
}

func (ski *SkinImg) analyseRegions() bool {
	img := *ski.Img
	bounds := img.Bounds()
	width := bounds.Size().X
	height := bounds.Size().Y

	skinRegions := ski.SkinRegions
	totalPixels := width * height
	skinRegionLen := len(skinRegions)

	// if there are less than 3 regions
	if skinRegionLen < 3 {
		//log.Println("Skin regions:" , skinRegionLen)
		return false
	}

	//Sort skin regions
	sort.Sort(sort.Reverse(skinRegions))

	//log.Println("Skin regions:", len(skinRegions))
	//Count total skin pixels
	totalSkin := 0
	for _, region := range skinRegions {
		totalSkin += region.Count
	}

	//log.Println("tk/tp", totalSkin, totalPixels)
	// check if there are more than 15% skin pixel in the image
	if (float64(totalSkin)/float64(totalPixels))*100 < 15 {
		// if the percentage lower than 15, it's not nude!
		log.Println("Skin ratio:", totalSkin, totalPixels, float64(totalSkin)/float64(totalPixels)*100.0)
		return false
	}

	// check if the largest skin region is less than 35% of the total skin count
	// AND if the second largest region is less than 30% of the total skin count
	// AND if the third largest region is less than 30% of the total skin count
	if (float64(skinRegions[0].Count)/float64(totalSkin))*100 < 35 && (float64(skinRegions[1].Count)/float64(totalSkin))*100 < 30 && (float64(skinRegions[2].Count)/float64(totalSkin))*100 < 30 {
		// the image is not nude.
		log.Println("it's not nude :) - less than 35%,30%,30% skin in the biggest areas :");
		return false
	}

	// check if the number of skin pixels in the largest region is less than 45% of the total skin count
	if (float64(skinRegions[0].Count)/float64(totalSkin))*100 < 45 {
		// it's not nude
		log.Println("it's not nude :) - the biggest region contains less than 45%: ", (float64(skinRegions[0].Count)/float64(totalSkin))*100);
		return false
	}

	if len(skinRegions) > 60 {
		log.Println("Skin region > 60")
		return false
	}

	return true
}

func IsNude(img *image.Image) bool {
	skimg := &SkinImg{img, Regions{}}
	return skimg.scanImage()
}
