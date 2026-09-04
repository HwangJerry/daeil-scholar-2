// AsciiGlobe — Responsive, motion-aware canvas globe for the landing hero
import { useEffect, useRef } from 'react';
import {
  createAsciiGlobePoints,
  createAsciiGlobeStars,
  getAsciiGlobeCharacter,
  isAsciiGlobeLandAt,
  type AsciiGlobePoint,
} from '../../lib/asciiGlobe';
import { cn } from '../../lib/utils';

const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)';
const TARGET_FRAME_RATE = 30;
const TARGET_FRAME_INTERVAL_MS = 1000 / TARGET_FRAME_RATE;
const ROTATION_SPEED_RADIANS_PER_SECOND = 0.18;
const INITIAL_ROTATION_Y = -Math.PI / 9;
const EARTH_AXIS_TILT_RADIANS = -Math.PI / 24;
const MAX_FRAME_DELTA_SECONDS = 0.1;
const MAX_DEVICE_PIXEL_RATIO = 2;
const GLOBE_RADIUS_RATIO = 0.4;
const COMPACT_CANVAS_BREAKPOINT_PX = 240;
const COMPACT_GRID_SIZE = 56;
const LARGE_GRID_SIZE = 76;
const COMPACT_FONT_RADIUS_DIVISOR = 22;
const LARGE_FONT_RADIUS_DIVISOR = 30;
const MIN_FONT_SIZE_PX = 3.4;
const STAR_FONT_SIZE_DIVISOR = 120;
const MIN_STAR_FONT_SIZE_PX = 2.5;
const AMBIENT_LIGHT = 0.13;
const LIGHT_CONTRAST_EXPONENT = 1.45;
const LAND_BASE_OPACITY = 0.34;
const OCEAN_BASE_OPACITY = 0.15;
const LAND_LIGHT_OPACITY = 0.66;
const OCEAN_LIGHT_OPACITY = 0.48;
const LIMB_OPACITY_FLOOR = 0.46;
const LIMB_FADE_DEPTH = 0.2;
const RADIANS_TO_DEGREES = 180 / Math.PI;

// Camera-space light: high and left, matching the reference's bright crescent
// while keeping enough frontal light for recognizable land masses.
const LIGHT_DIRECTION = { x: -0.5, y: -0.32, z: 0.8 };

const COMPACT_GLOBE_POINTS = createAsciiGlobePoints(COMPACT_GRID_SIZE);
const LARGE_GLOBE_POINTS = createAsciiGlobePoints(LARGE_GRID_SIZE);
const BACKGROUND_STARS = createAsciiGlobeStars();

interface CanvasDimensions {
  width: number;
  height: number;
}

interface AsciiGlobeProps {
  className?: string;
}

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value));
}

function smoothstep(edgeStart: number, edgeEnd: number, value: number) {
  const normalizedValue = clamp(
    (value - edgeStart) / (edgeEnd - edgeStart),
    0,
    1,
  );

  return normalizedValue * normalizedValue * (3 - 2 * normalizedValue);
}

function getSurfaceLocation(point: AsciiGlobePoint, rotationY: number) {
  const tiltCosine = Math.cos(EARTH_AXIS_TILT_RADIANS);
  const tiltSine = Math.sin(EARTH_AXIS_TILT_RADIANS);
  const untiltedY =
    point.normalizedY * tiltCosine + point.normalizedZ * tiltSine;
  const untiltedZ =
    -point.normalizedY * tiltSine + point.normalizedZ * tiltCosine;
  const latitudeDegrees = Math.asin(untiltedY) * RADIANS_TO_DEGREES;
  const longitudeDegrees =
    (Math.atan2(untiltedZ, point.normalizedX) - rotationY) *
    RADIANS_TO_DEGREES;

  return { latitudeDegrees, longitudeDegrees };
}

function drawBackgroundStars(
  context: CanvasRenderingContext2D,
  dimensions: CanvasDimensions,
  color: string,
) {
  const { width, height } = dimensions;
  const fontSize = Math.max(
    MIN_STAR_FONT_SIZE_PX,
    Math.min(width, height) / STAR_FONT_SIZE_DIVISOR,
  );

  context.fillStyle = color;
  context.font = `${fontSize}px monospace`;

  for (const star of BACKGROUND_STARS) {
    context.globalAlpha = star.opacity;
    context.fillText(
      star.character,
      width / 2 + star.normalizedX * width * 0.49,
      height / 2 + star.normalizedY * height * 0.49,
    );
  }
}

function drawGlobe(
  canvas: HTMLCanvasElement,
  context: CanvasRenderingContext2D,
  dimensions: CanvasDimensions,
  rotationY: number,
) {
  const { width, height } = dimensions;
  const isCompactCanvas = Math.min(width, height) < COMPACT_CANVAS_BREAKPOINT_PX;
  const points = isCompactCanvas ? COMPACT_GLOBE_POINTS : LARGE_GLOBE_POINTS;
  const fontRadiusDivisor = isCompactCanvas
    ? COMPACT_FONT_RADIUS_DIVISOR
    : LARGE_FONT_RADIUS_DIVISOR;
  const radius = Math.min(width, height) * GLOBE_RADIUS_RATIO;
  const centerX = width / 2;
  const centerY = height / 2;
  const computedStyle = getComputedStyle(canvas);
  const surfaceColor =
    computedStyle.getPropertyValue('--color-surface').trim() ||
    computedStyle.color;
  const fontSize = Math.max(MIN_FONT_SIZE_PX, radius / fontRadiusDivisor);

  context.clearRect(0, 0, width, height);
  context.textAlign = 'center';
  context.textBaseline = 'middle';
  drawBackgroundStars(context, dimensions, surfaceColor);

  context.fillStyle = surfaceColor;
  context.font = `${computedStyle.fontWeight} ${fontSize}px ${computedStyle.fontFamily}`;

  for (const point of points) {
    const lightDot = Math.max(
      0,
      point.normalizedX * LIGHT_DIRECTION.x +
        point.normalizedY * LIGHT_DIRECTION.y +
        point.normalizedZ * LIGHT_DIRECTION.z,
    );
    const brightness =
      AMBIENT_LIGHT +
      lightDot ** LIGHT_CONTRAST_EXPONENT * (1 - AMBIENT_LIGHT);
    const { latitudeDegrees, longitudeDegrees } = getSurfaceLocation(
      point,
      rotationY,
    );
    const isLand = isAsciiGlobeLandAt(latitudeDegrees, longitudeDegrees);
    const character = getAsciiGlobeCharacter(
      isLand,
      brightness,
      point.textureNoise,
    );
    const surfaceOpacity = isLand
      ? LAND_BASE_OPACITY + brightness * LAND_LIGHT_OPACITY
      : OCEAN_BASE_OPACITY + brightness * OCEAN_LIGHT_OPACITY;
    const limbOpacity =
      LIMB_OPACITY_FLOOR +
      smoothstep(0, LIMB_FADE_DEPTH, point.normalizedZ) *
        (1 - LIMB_OPACITY_FLOOR);

    context.globalAlpha = surfaceOpacity * limbOpacity;
    context.fillText(
      character,
      centerX + point.normalizedX * radius,
      centerY + point.normalizedY * radius,
    );
  }

  context.globalAlpha = 1;
}

export function AsciiGlobe({ className }: AsciiGlobeProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || typeof window.requestAnimationFrame !== 'function') return;

    let context: CanvasRenderingContext2D | null = null;
    let animationFrameId: number | null = null;
    let lastFrameTime: number | null = null;
    let rotationY = INITIAL_ROTATION_Y;
    let isInViewport = true;
    let isDocumentVisible = document.visibilityState === 'visible';
    let dimensions: CanvasDimensions = { width: 0, height: 0 };
    const reducedMotionQuery =
      typeof window.matchMedia === 'function'
        ? window.matchMedia(REDUCED_MOTION_QUERY)
        : null;
    let prefersReducedMotion = reducedMotionQuery?.matches ?? false;

    const renderFrame = () => {
      if (!context || dimensions.width === 0 || dimensions.height === 0) return;
      drawGlobe(canvas, context, dimensions, rotationY);
    };

    const stopAnimation = () => {
      if (animationFrameId === null) return;
      window.cancelAnimationFrame(animationFrameId);
      animationFrameId = null;
    };

    const animate = (frameTime: number) => {
      animationFrameId = null;
      if (prefersReducedMotion || !isDocumentVisible || !isInViewport) return;

      const previousFrameTime = lastFrameTime;
      const isFirstFrame = previousFrameTime === null;
      const elapsedMilliseconds =
        previousFrameTime === null
          ? TARGET_FRAME_INTERVAL_MS
          : frameTime - previousFrameTime;
      const shouldRenderFrame =
        isFirstFrame || elapsedMilliseconds >= TARGET_FRAME_INTERVAL_MS;

      if (shouldRenderFrame) {
        const elapsedSeconds = Math.min(
          elapsedMilliseconds / 1000,
          MAX_FRAME_DELTA_SECONDS,
        );
        rotationY += ROTATION_SPEED_RADIANS_PER_SECOND * elapsedSeconds;
        lastFrameTime = frameTime;
        renderFrame();
      }

      animationFrameId = window.requestAnimationFrame(animate);
    };

    const syncAnimation = () => {
      stopAnimation();
      lastFrameTime = null;

      if (prefersReducedMotion) {
        if (isDocumentVisible && isInViewport) renderFrame();
        return;
      }
      if (!isDocumentVisible || !isInViewport) return;

      animationFrameId = window.requestAnimationFrame(animate);
    };

    const resizeCanvas = () => {
      const bounds = canvas.getBoundingClientRect();
      const width = Math.round(bounds.width);
      const height = Math.round(bounds.height);
      if (width === 0 || height === 0) return;

      context ??= canvas.getContext('2d');
      if (!context) return;

      const pixelRatio = Math.min(
        window.devicePixelRatio || 1,
        MAX_DEVICE_PIXEL_RATIO,
      );
      canvas.width = Math.round(width * pixelRatio);
      canvas.height = Math.round(height * pixelRatio);
      context.setTransform(pixelRatio, 0, 0, pixelRatio, 0, 0);
      dimensions = { width, height };
      renderFrame();
    };

    const handleVisibilityChange = () => {
      isDocumentVisible = document.visibilityState === 'visible';
      syncAnimation();
    };
    const handleReducedMotionChange = (event: MediaQueryListEvent) => {
      prefersReducedMotion = event.matches;
      syncAnimation();
    };

    const resizeObserver =
      typeof ResizeObserver === 'undefined'
        ? null
        : new ResizeObserver(resizeCanvas);
    const intersectionObserver =
      typeof IntersectionObserver === 'undefined'
        ? null
        : new IntersectionObserver((entries) => {
            isInViewport = entries.some((entry) => entry.isIntersecting);
            syncAnimation();
          });

    resizeObserver?.observe(canvas);
    intersectionObserver?.observe(canvas);
    if (!resizeObserver) window.addEventListener('resize', resizeCanvas);
    document.addEventListener('visibilitychange', handleVisibilityChange);
    reducedMotionQuery?.addEventListener('change', handleReducedMotionChange);

    resizeCanvas();
    syncAnimation();

    return () => {
      stopAnimation();
      resizeObserver?.disconnect();
      intersectionObserver?.disconnect();
      if (!resizeObserver) window.removeEventListener('resize', resizeCanvas);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      reducedMotionQuery?.removeEventListener('change', handleReducedMotionChange);
    };
  }, []);

  return (
    <div
      aria-hidden="true"
      className={cn(
        'relative size-[180px] shrink-0 sm:size-[210px] md:size-[340px] lg:size-[390px]',
        className,
      )}
    >
      <canvas
        ref={canvasRef}
        className={cn(
          'pointer-events-none block size-full select-none font-mono font-medium text-surface',
        )}
      />
    </div>
  );
}
