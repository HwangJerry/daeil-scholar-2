// asciiGlobe — Pure sampling and character-selection helpers for the landing globe
import { WORLD_LAND_MASK } from '../constants/worldLandMask';

const DEFAULT_GLOBE_GRID_SIZE = 76;
const DEFAULT_STAR_COUNT = 84;
const FULL_LATITUDE_DEGREES = 180;
const FULL_LONGITUDE_DEGREES = 360;
const NORTH_POLE_DEGREES = 90;
const WESTERN_EDGE_DEGREES = -180;
const MAX_NORMALIZED_POINT_RADIUS = 0.985;
const MAX_NORMALIZED_POINT_RADIUS_SQUARED =
  MAX_NORMALIZED_POINT_RADIUS * MAX_NORMALIZED_POINT_RADIUS;
const POINT_JITTER_RATIO = 0.34;
const NORMALIZED_DIAMETER = 2;
const NOISE_X_FACTOR = 12.9898;
const NOISE_Y_FACTOR = 78.233;
const NOISE_SEED_FACTOR = 37.719;
const NOISE_SCALE = 43_758.5453;
const LAND_BRIGHTNESS_BOOST = 0.16;
const MINIMUM_BRIGHTNESS = 0;
const MAXIMUM_BRIGHTNESS = 1;

const LAND_CHARACTERS = ['.', ':', '+', 'x', '*', '#', '@'] as const;
const OCEAN_CHARACTERS = ['.', '.', ':', '-', '+', 'x'] as const;
const STAR_CHARACTERS = ['.', '.', '·', '+'] as const;

export interface AsciiGlobePoint {
  normalizedX: number;
  normalizedY: number;
  normalizedZ: number;
  textureNoise: number;
}

export interface AsciiGlobeStar {
  normalizedX: number;
  normalizedY: number;
  opacity: number;
  character: (typeof STAR_CHARACTERS)[number];
}

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value));
}

function createDeterministicNoise(x: number, y: number, seed: number) {
  const noise =
    Math.sin(
      x * NOISE_X_FACTOR +
        y * NOISE_Y_FACTOR +
        seed * NOISE_SEED_FACTOR,
    ) * NOISE_SCALE;

  return noise - Math.floor(noise);
}

function findNearestMaskIndex(
  value: number,
  minimum: number,
  range: number,
  cellCount: number,
) {
  const normalizedValue = (value - minimum) / range;
  const nearestIndex = Math.round(normalizedValue * (cellCount - 1));

  return clamp(nearestIndex, 0, cellCount - 1);
}

export function isAsciiGlobeLandAt(
  latitudeDegrees: number,
  longitudeDegrees: number,
) {
  const wrappedLongitude =
    ((((longitudeDegrees - WESTERN_EDGE_DEGREES) % FULL_LONGITUDE_DEGREES) +
      FULL_LONGITUDE_DEGREES) %
      FULL_LONGITUDE_DEGREES) +
    WESTERN_EDGE_DEGREES;
  const rowIndex = findNearestMaskIndex(
    NORTH_POLE_DEGREES - latitudeDegrees,
    0,
    FULL_LATITUDE_DEGREES,
    WORLD_LAND_MASK.length,
  );
  const columnIndex = findNearestMaskIndex(
    wrappedLongitude,
    WESTERN_EDGE_DEGREES,
    FULL_LONGITUDE_DEGREES,
    WORLD_LAND_MASK[rowIndex].length,
  );

  return WORLD_LAND_MASK[rowIndex][columnIndex] === '1';
}

export function createAsciiGlobePoints(
  gridSize = DEFAULT_GLOBE_GRID_SIZE,
): readonly AsciiGlobePoint[] {
  const cellSize = NORMALIZED_DIAMETER / gridSize;
  const points: AsciiGlobePoint[] = [];

  for (let rowIndex = 0; rowIndex < gridSize; rowIndex += 1) {
    for (let columnIndex = 0; columnIndex < gridSize; columnIndex += 1) {
      const xNoise = createDeterministicNoise(columnIndex, rowIndex, 1);
      const yNoise = createDeterministicNoise(columnIndex, rowIndex, 2);
      const normalizedX =
        -1 +
        (columnIndex + 0.5) * cellSize +
        (xNoise - 0.5) * cellSize * POINT_JITTER_RATIO;
      const normalizedY =
        -1 +
        (rowIndex + 0.5) * cellSize +
        (yNoise - 0.5) * cellSize * POINT_JITTER_RATIO;
      const distanceFromCenterSquared =
        normalizedX * normalizedX + normalizedY * normalizedY;

      if (distanceFromCenterSquared > MAX_NORMALIZED_POINT_RADIUS_SQUARED) {
        continue;
      }

      points.push({
        normalizedX,
        normalizedY,
        normalizedZ: Math.sqrt(1 - distanceFromCenterSquared),
        textureNoise: createDeterministicNoise(columnIndex, rowIndex, 3),
      });
    }
  }

  return points;
}

export function createAsciiGlobeStars(
  starCount = DEFAULT_STAR_COUNT,
): readonly AsciiGlobeStar[] {
  return Array.from({ length: starCount }, (_, starIndex) => {
    const characterNoise = createDeterministicNoise(starIndex, 0, 6);
    const characterIndex = Math.min(
      STAR_CHARACTERS.length - 1,
      Math.floor(characterNoise * STAR_CHARACTERS.length),
    );

    return {
      normalizedX: createDeterministicNoise(starIndex, 0, 4) * 2 - 1,
      normalizedY: createDeterministicNoise(starIndex, 0, 5) * 2 - 1,
      opacity: 0.12 + createDeterministicNoise(starIndex, 0, 7) * 0.3,
      character: STAR_CHARACTERS[characterIndex],
    };
  });
}

export function getAsciiGlobeCharacter(
  isLand: boolean,
  brightness: number,
  textureNoise = 0.5,
) {
  const characters = isLand ? LAND_CHARACTERS : OCEAN_CHARACTERS;
  const surfaceBrightness = clamp(
    brightness + (isLand ? LAND_BRIGHTNESS_BOOST : 0),
    MINIMUM_BRIGHTNESS,
    MAXIMUM_BRIGHTNESS,
  );
  const texturedBrightness = clamp(
    surfaceBrightness * (0.82 + textureNoise * 0.26),
    MINIMUM_BRIGHTNESS,
    MAXIMUM_BRIGHTNESS,
  );
  const characterIndex = Math.min(
    characters.length - 1,
    Math.floor(texturedBrightness * characters.length),
  );

  return characters[characterIndex];
}
