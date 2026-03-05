package ui

import (
	_ "embed"
	"fmt"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed shaders/bgr15.kage
var bgr15ShaderSource []byte

//go:embed shaders/crt_basic.kage
var crtBasicShaderSource []byte

type Shader interface {
	GetUniforms() map[string]any
	GetShader() *ebiten.Shader
}

type Bgr15 struct {
	shader *ebiten.Shader
}

func (s *Bgr15) GetUniforms() map[string]any {
	return nil
}

func (s *Bgr15) GetShader() *ebiten.Shader {
	return s.shader
}

func NewBgr15Shader() (Shader, error) {
	shader, err := ebiten.NewShader(bgr15ShaderSource)
	if err != nil {
		return nil, fmt.Errorf("Kage: Shader compilation failed. " + err.Error())
	}

	s := &Bgr15{shader: shader}
	return s, nil
}

type CrtBasic struct {
	shader   *ebiten.Shader
	uniforms map[string]any
	tick     float64
}

func NewCrtBasicShader() (Shader, error) {
	shader, err := ebiten.NewShader(crtBasicShaderSource)
	if err != nil {
		return nil, fmt.Errorf("Kage: Shader compilation failed. " + err.Error())
	}

	s := &CrtBasic{
		shader:   shader,
		uniforms: make(map[string]any),
	}
	return s, nil
}

func (s *CrtBasic) GetUniforms() map[string]any {
	s.tick += 1 / 60.0

	s.uniforms["Seed"] = int32(rand.Int32N(15_000))
	s.uniforms["Tick"] = float32(s.tick)

	return s.uniforms
}

func (s *CrtBasic) GetShader() *ebiten.Shader {
	return s.shader
}
