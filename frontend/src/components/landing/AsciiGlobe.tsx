// AsciiGlobe — Responsive, motion-aware canvas globe for the landing hero
import { useEffect, useRef } from 'react';
import {
  createAsciiGlobePoints,
  getAsciiGlobeCharacter,
} from '../../lib/asciiGlobe';
import { cn } from '../../lib/utils';

const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)';
const ROTATION_SPEED_RADIANS_PER_SECOND = 0.12;
const INITIAL_ROTATION_Y = -Math.PI / 6;
const MAX_FRAME_DELTA_SECONDS = 0.1;
const MAX_DEVICE_PIXEL_RATIO = 2;
const GLOBE_RADIUS_RATIO = 0.46;
const MIN_FONT_SIZE_PX = 6;
const FONT_RADIUS_DIVISOR = 17;
const MIN_POINT_OPACITY = 0.45;

const GLOBE_POINTS = createAsciiGlobePoints();

interface CanvasDimensions {
  width: number;
  height: number;
}

interface AsciiGlobeProps {
  className?: string;
}

function drawGlobe(
  canvas: HTMLCanvasElement,
  context: CanvasRenderingContext2D,
  dimensions: CanvasDimensions,
  rotationY: number,
) {
  const { width, height } = dimensions;
  const radius = Math.min(width, height) * GLOBE_RADIUS_RATIO;
  const centerX = width / 2;
  const centerY = height / 2;
  const computedStyle = getComputedStyle(canvas);
  const surfaceColor = computedStyle.getPropertyValue('--color-surface').trim();
  const fontSize = Math.max(MIN_FONT_SIZE_PX, radius / FONT_RADIUS_DIVISOR);

  context.clearRect(0, 0, width, height);
  context.fillStyle = surfaceColor || computedStyle.color;
  context.font = `${computedStyle.fontWeight} ${fontSize}px ${computedStyle.fontFamily}`;
  context.textAlign = 'center';
  context.textBaseline = 'middle';

  for (const point of GLOBE_POINTS) {
    const rotatedLongitude = point.longitudeRadians + rotationY;
    const x = point.cosLatitude * Math.cos(rotatedLongitude);
    const z = point.cosLatitude * Math.sin(rotatedLongitude);
    if (z <= 0) continue;

    const character = getAsciiGlobeCharacter(point.isLand, z);
    if (!character) continue;

    context.globalAlpha = MIN_POINT_OPACITY + z * (1 - MIN_POINT_OPACITY);
    context.fillText(
      character,
      centerX + x * radius,
      centerY - point.sinLatitude * radius,
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

      if (lastFrameTime !== null) {
        const elapsedSeconds = Math.min(
          (frameTime - lastFrameTime) / 1000,
          MAX_FRAME_DELTA_SECONDS,
        );
        rotationY += ROTATION_SPEED_RADIANS_PER_SECOND * elapsedSeconds;
      }
      lastFrameTime = frameTime;
      renderFrame();
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
