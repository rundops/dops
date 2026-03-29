<script setup>
import { onMounted, onUnmounted, ref } from 'vue'

const canvasRef = ref(null)
let animId = 0

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')

  const img = new Image()
  img.src = '/logo.png'

  const dpr = window.devicePixelRatio || 1
  const W = 380
  const H = 380
  canvas.width = W * dpr
  canvas.height = H * dpr
  ctx.scale(dpr, dpr)

  class Sparkle {
    constructor() { this.reset() }
    reset() {
      this.x = (Math.random() - 0.5) * 300
      this.y = (Math.random() - 0.5) * 300
      this.size = Math.random() * 2.5 + 0.8
      this.life = 0
      this.maxLife = Math.random() * 120 + 60
      this.speed = Math.random() * 0.3 + 0.1
      this.angle = Math.random() * Math.PI * 2
      this.hue = Math.random() * 60 + 240
    }
    update() {
      this.life++
      this.x += Math.cos(this.angle) * this.speed
      this.y += Math.sin(this.angle) * this.speed - 0.2
      if (this.life >= this.maxLife) this.reset()
    }
    draw(cx, cy) {
      const alpha = 1 - this.life / this.maxLife
      const pulse = 0.5 + 0.5 * Math.sin(this.life * 0.15)
      const s = this.size * (0.8 + pulse * 0.4)
      ctx.save()
      ctx.globalAlpha = alpha * 0.8
      ctx.fillStyle = `hsl(${this.hue}, 80%, 75%)`
      ctx.shadowColor = `hsl(${this.hue}, 90%, 70%)`
      ctx.shadowBlur = 8
      const x = cx + this.x, y = cy + this.y
      ctx.beginPath()
      for (let i = 0; i < 4; i++) {
        const a = (i / 4) * Math.PI * 2 - Math.PI / 2
        const aNext = ((i + 0.5) / 4) * Math.PI * 2 - Math.PI / 2
        ctx.lineTo(x + Math.cos(a) * s * 2.5, y + Math.sin(a) * s * 2.5)
        ctx.lineTo(x + Math.cos(aNext) * s * 0.6, y + Math.sin(aNext) * s * 0.6)
      }
      ctx.closePath()
      ctx.fill()
      ctx.restore()
    }
  }

  const sparkles = Array.from({ length: 24 }, () => new Sparkle())
  let t = 0

  function draw() {
    animId = requestAnimationFrame(draw)
    ctx.clearRect(0, 0, W, H)
    t++

    const bobY = Math.sin(t * 0.018) * 8
    const swayX = Math.sin(t * 0.011) * 3
    const tilt = Math.sin(t * 0.015) * 0.025
    const breathe = 1 + Math.sin(t * 0.022) * 0.012

    const cx = W / 2 + swayX
    const cy = H / 2 + bobY

    // Glow
    const glowPulse = 0.25 + 0.12 * Math.sin(t * 0.04)
    const gradient = ctx.createRadialGradient(cx, cy, 20, cx, cy, 180)
    gradient.addColorStop(0, `rgba(147, 100, 255, ${glowPulse})`)
    gradient.addColorStop(0.5, `rgba(100, 60, 220, ${glowPulse * 0.4})`)
    gradient.addColorStop(1, 'rgba(100, 60, 220, 0)')
    ctx.fillStyle = gradient
    ctx.fillRect(0, 0, W, H)

    // Sparkles behind
    sparkles.slice(0, 12).forEach(s => { s.update(); s.draw(cx, cy) })

    // Cat
    if (img.complete && img.naturalWidth > 0) {
      ctx.save()
      ctx.translate(cx, cy)
      ctx.rotate(tilt)
      ctx.scale(breathe, breathe)

      const size = 300
      ctx.drawImage(img, -size / 2, -size / 2, size, size)

      // Collar gem glow
      const gemGlow = 0.4 + 0.4 * Math.sin(t * 0.06)
      ctx.globalAlpha = gemGlow
      ctx.fillStyle = '#7df9ff'
      ctx.shadowColor = '#7df9ff'
      ctx.shadowBlur = 20 + 10 * Math.sin(t * 0.06)
      ctx.beginPath()
      ctx.arc(0, size * 0.13, size * 0.015, 0, Math.PI * 2)
      ctx.fill()

      ctx.restore()
    }

    // Sparkles in front
    sparkles.slice(12).forEach(s => { s.update(); s.draw(cx, cy) })
  }

  img.onload = draw
  if (img.complete) draw()
})

onUnmounted(() => {
  cancelAnimationFrame(animId)
})
</script>

<template>
  <canvas ref="canvasRef" class="hero-animation" />
</template>

<style scoped>
.hero-animation {
  width: 320px;
  height: 320px;
  display: block;
}

@media (min-width: 640px) {
  .hero-animation {
    width: 380px;
    height: 380px;
  }
}
</style>
