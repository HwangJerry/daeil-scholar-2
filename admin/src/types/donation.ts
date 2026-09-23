// Donation API types — public summary, admin display configuration and snapshots.
export interface DonationConfig {
  dcBalanceAmount: number | null;
  dcBalanceAsOf: string | null;
  dcSeq: number;
  dcGoal: number;
  dcManualAdj: number;
  dcManualDonorCnt: number;
  dcTierSproutMin: number;
  dcTierSaplingMin: number;
  dcTierTreeMin: number;
  dcTierBloomingMin: number;
  dcTierFruitingMin: number;
  dcNote: string;
  dcOverwrite: string; // "Y" | "N"
  isActive: string;
  regDate: string;
}

export interface DonationConfigUpdateRequest {
  balanceAmount: number | null;
  balanceAsOf: string | null;
  goal: number;
  manualAdj: number;
  manualDonorCnt: number;
  tierSproutMin: number;
  tierSaplingMin: number;
  tierTreeMin: number;
  tierBloomingMin: number;
  tierFruitingMin: number;
  note: string;
  overwrite: boolean;
}

export interface DonationSnapshot {
  dsDate: string;
  dsTotal: number;
  dsManualAdj: number;
  dsDonorCnt: number;
  dsGoal: number;
}

export interface DonationSummary {
  monthAmount: number;
  balanceAmount: number | null;
  balanceAsOf: string | null;
  displayAmount: number;
  goalAmount: number;
  donorCount: number;
  achievementRate: number;
  snapshotDate: string;
  tierThresholds: {
    sprout: number;
    sapling: number;
    tree: number;
    blooming: number;
    fruiting: number;
  };
}
