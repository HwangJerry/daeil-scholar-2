// Dashboard statistics card — displays a metric with label, value, icon, and optional subtext
import type React from "react";

interface StatsCardProps {
  label: string;
  value: string | number;
  icon?: React.ReactNode;
  subtext?: string;
}

export function StatsCard({ label, value, icon, subtext }: StatsCardProps) {
  return (
    <div className="bg-white rounded-2xl border border-slate-100 p-5 shadow-sm">
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-cool-gray">{label}</span>
        {icon && <span className="text-cool-gray">{icon}</span>}
      </div>
      <p className="text-2xl font-bold text-dark-slate">{value}</p>
      {subtext && <p className="text-xs text-cool-gray mt-1">{subtext}</p>}
    </div>
  );
}
