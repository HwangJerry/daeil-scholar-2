// asciiGlobe — Pure sampling and character-selection helpers for the landing globe
import { WORLD_LAND_MASK } from '../constants/worldLandMask';

const LATITUDE_STEPS = 48;
const LONGITUDE_STEPS = 96;
const FULL_LATITUDE_DEGREES = 180;
const FULL_LONGITUDE_DEGREES = 360;
const NORTH_POLE_DEGREES = 90;
const WESTERN_EDGE_DEGREES = -180;
const DEGREES_TO_RADIANS = Math.PI / 180;

const LAND_LOW_BRIGHTNESS_LIMIT = 0.25;
const LAND_MEDIUM_BRIGHTNESS_LIMIT = 0.5;
const LAND_HIGH_BRIGHTNESS_LIMIT = 0.75;

// Ocean is shown too (not just land) so the point cloud reads as a full sphere
// rather than floating continents — the low limit sits well below land's so
// most of the visible hemisphere still carries a faint '·' texture.
const OCEAN_LOW_BRIGHTNESS_LIMIT = 0.1;
const OCEAN_MEDIUM_BRIGHTNESS_LIMIT = 0.4;
const OCEAN_HIGH_BRIGHTNESS_LIMIT = 0.7;

export interface AsciiGlobePoint {
  cosLatitude: number;
  sinLatitude: number;
  longitudeRadians: number;
  isLand: boolean;
}

function findNearestMaskIndex(
  value: number,
  minimum: number,
  range: number,
  cellCount: number,
) {
  const normalizedValue = (value - minimum) / range;
  return Math.round(normalizedValue * (cellCount - 1));
}

function isLandAt(latitudeDegrees: number, longitudeDegrees: number) {
  const rowIndex = findNearestMaskIndex(
    NORTH_POLE_DEGREES - latitudeDegrees,
    0,
    FULL_LATITUDE_DEGREES,
    WORLD_LAND_MASK.length,
  );
  const columnIndex = findNearestMaskIndex(
    longitudeDegrees,
    WESTERN_EDGE_DEGREES,
    FULL_LONGITUDE_DEGREES,
    WORLD_LAND_MASK[rowIndex].length,
  );

  return WORLD_LAND_MASK[rowIndex][columnIndex] === '1';
}

export function createAsciiGlobePoints(): readonly AsciiGlobePoint[] {
  return Array.from({ length: LATITUDE_STEPS }, (_, latitudeIndex) => {
    const latitudeDegrees =
      -NORTH_POLE_DEGREES +
      ((latitudeIndex + 0.5) / LATITUDE_STEPS) * FULL_LATITUDE_DEGREES;
    const latitudeRadians = latitudeDegrees * DEGREES_TO_RADIANS;

    return Array.from({ length: LONGITUDE_STEPS }, (_, longitudeIndex) => {
      const longitudeDegrees =
        WESTERN_EDGE_DEGREES +
        (longitudeIndex / LONGITUDE_STEPS) * FULL_LONGITUDE_DEGREES;

      return {
        cosLatitude: Math.cos(latitudeRadians),
        sinLatitude: Math.sin(latitudeRadians),
        longitudeRadians: longitudeDegrees * DEGREES_TO_RADIANS,
        isLand: isLandAt(latitudeDegrees, longitudeDegrees),
      };
    });
  }).flat();
}

export function getAsciiGlobeCharacter(isLand: boolean, brightness: number) {
  if (isLand) {
    if (brightness < LAND_LOW_BRIGHTNESS_LIMIT) return '·';
    if (brightness < LAND_MEDIUM_BRIGHTNESS_LIMIT) return 'x';
    if (brightness < LAND_HIGH_BRIGHTNESS_LIMIT) return '#';
    return '@';
  }

  if (brightness < OCEAN_LOW_BRIGHTNESS_LIMIT) return '';
  if (brightness < OCEAN_MEDIUM_BRIGHTNESS_LIMIT) return '·';
  if (brightness < OCEAN_HIGH_BRIGHTNESS_LIMIT) return 'x';
  return 'X';
}
