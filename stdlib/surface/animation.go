package surface

type Animation struct {
	frames            []*Image
	speed             int
	frameTime         int
	currentFrameIndex int
}

type AnimatedSprite struct {
	currentAnimationName string
	animations           map[string]*Animation
}

func NewAnimatedSprite() *AnimatedSprite {
	return &AnimatedSprite{
		animations: make(map[string]*Animation),
	}
}

func (sprite *AnimatedSprite) RegisterAnimation(name string, frames []*Image, speed int) {
	if sprite == nil || name == "" || len(frames) == 0 {
		return
	}
	if speed < 0 {
		speed = 0
	}
	sprite.animations[name] = &Animation{
		frames: frames,
		speed:  speed,
	}
}

func (sprite *AnimatedSprite) Play(name string) {
	if sprite == nil {
		return
	}
	animation, ok := sprite.animations[name]
	if !ok || animation == nil || len(animation.frames) == 0 {
		return
	}
	sprite.currentAnimationName = name
	animation.frameTime++
	if animation.frameTime > animation.speed {
		animation.currentFrameIndex = (animation.currentFrameIndex + 1) % len(animation.frames)
		animation.frameTime = 0
	}
}

func (sprite *AnimatedSprite) Current() *Image {
	if sprite == nil || len(sprite.animations) == 0 {
		return nil
	}
	if sprite.currentAnimationName == "" {
		for name := range sprite.animations {
			sprite.currentAnimationName = name
			break
		}
	}
	animation := sprite.animations[sprite.currentAnimationName]
	if animation == nil || len(animation.frames) == 0 {
		return nil
	}
	return animation.frames[animation.currentFrameIndex]
}
