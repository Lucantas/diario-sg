export interface MonthlyDonations {
  month: string;
  goal_cents: number;
  received_cents: number;
  updated_on: string;
}

export interface GoalProgress {
  percent: number;
  received: string;
  goal: string;
  month: string;
  updatedOn: string;
  reached: boolean;
}

function reais(cents: number): string {
  const whole = cents % 100 === 0;
  const value = (cents / 100).toLocaleString("pt-BR", {
    minimumFractionDigits: whole ? 0 : 2,
    maximumFractionDigits: whole ? 0 : 2,
  });
  return `R$ ${value}`;
}

function calendarDate(isoDate: string): Date {
  const [year, month, day = 1] = isoDate.split("-").map(Number);
  return new Date(year, month - 1, day);
}

export function goalProgress(d: MonthlyDonations): GoalProgress {
  return {
    percent: Math.min(100, Math.floor((d.received_cents / d.goal_cents) * 100)),
    received: reais(d.received_cents),
    goal: reais(d.goal_cents),
    month: calendarDate(d.month).toLocaleDateString("pt-BR", { month: "long", year: "numeric" }),
    updatedOn: calendarDate(d.updated_on).toLocaleDateString("pt-BR", { day: "numeric", month: "long" }),
    reached: d.received_cents >= d.goal_cents,
  };
}
