// Shared API types. These mirror the Go DTOs; keep them in sync with the
// backend handlers.

export interface User {
  id: string;
  email: string;
  displayName: string;
  heightCm: number | null;
  sex: "male" | "female" | "other" | null;
  telegramUsername: string | null;
  emailVerified: boolean;
  createdAt: string;
}

export interface ApiError {
  code: string;
  message: string;
  fields?: Record<string, string>;
}

// --- water ---

export type WaterSource = "glass" | "bottle_small" | "bottle_large" | "custom";

export interface WaterIntake {
  id: string;
  amountMl: number;
  source: WaterSource;
  consumedAt: string;
}

export interface WaterDay {
  date: string;
  goalMl: number;
  consumedMl: number;
  remainingMl: number;
  percent: number;
  intakes: WaterIntake[];
}

export interface WaterHistory {
  goalMl: number;
  series: { date: string; totalMl: number }[];
}

// --- weight ---

export interface WeightEntry {
  id: string;
  weightKg: number;
  note: string;
  measuredOn: string;
  measuredAt: string;
}

export interface WeightSummary {
  latestKg: number | null;
  targetKg: number | null;
  heightCm: number | null;
  bmi: number | null;
  bmiCategory: "" | "underweight" | "normal" | "overweight" | "obese";
  series: WeightEntry[];
}

// --- nutrition ---

export type Meal = "breakfast" | "lunch" | "dinner" | "snack";

export interface Macros {
  kcal: number;
  protein: number;
  fat: number;
  carbs: number;
}

export interface NutritionGoal {
  kcal: number;
  protein: number;
  fat: number;
  carbs: number;
}

export interface DietEntry {
  id: string;
  dishId: string | null;
  name: string;
  grams: number | null;
  meal: Meal | null;
  source: "manual" | "dish";
  macros: Macros;
  consumedAt: string;
}

export interface Activity {
  id: string;
  type: string;
  kcal: number;
  durationMin: number | null;
  met: number | null;
  source: "manual" | "met";
  performedAt: string;
}

export interface DaySummary {
  date: string;
  goal: NutritionGoal;
  consumed: Macros;
  burnedKcal: number;
  netKcal: number;
  remainingKcal: number;
  entries: DietEntry[];
  activities: Activity[];
}

export interface DayTotals {
  date: string;
  kcal: number;
  protein: number;
  fat: number;
  carbs: number;
  burnedKcal: number;
}

export interface ActivityType {
  key: string;
  label: string;
  met: number;
}

export interface Dish {
  id: string;
  name: string;
  description: string;
  recipe: string;
  imageUrl: string | null;
  per100g: Macros;
  servingGrams: number | null;
  ratingAvg: number;
  ratingCount: number;
  myRating: number | null;
  isFavorite: boolean;
  isMine: boolean;
  authorName: string;
  createdAt: string;
}

export interface DishComment {
  id: string;
  authorName: string;
  body: string;
  createdAt: string;
  isMine: boolean;
}

export type DishScope = "all" | "mine" | "favorites";
export type DishSort = "new" | "rating" | "name";
