package nec

const (
	Power            uint32 = 0x00F7C03F
	BrightnessUp     uint32 = 0x00F700FF
	BrightnessDown   uint32 = 0x00F7807F
	PlayPause        uint32 = 0x00F740BF
	Stop             uint32 = 0x00F7C837
	Next             uint32 = 0x00F7A05F
	Previous         uint32 = 0x00F7609F

	Red              uint32 = 0x00F720DF
	Green            uint32 = 0x00F7A857
	Blue             uint32 = 0x00F750AF
	White            uint32 = 0x00F7E01F

	Color1           uint32 = 0x00F710EF
	Color2           uint32 = 0x00F7906F
	Color3           uint32 = 0x00F730CF
	Color4           uint32 = 0x00F7B04F
	Color5           uint32 = 0x00F708F7
	Color6           uint32 = 0x00F78877
	Color7           uint32 = 0x00F728D7
	Color8           uint32 = 0x00F7A857
	Color9           uint32 = 0x00F704FB
	Color0           uint32 = 0x00F7847B
	OK               uint32 = 0x00F724DB
	Back             uint32 = 0x00F7A45B

	EffectFlash      uint32 = PlayPause
	EffectFade       uint32 = Stop
	EffectSmooth     uint32 = Next
	EffectStrobe     uint32 = Previous
)