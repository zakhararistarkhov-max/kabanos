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

export interface DishIngredient {
  dishId: string;
  name: string;
  grams: number;
  per100g: Macros;
  contribution: Macros;
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
  isPublic: boolean;
  authorName: string;
  createdAt: string;
  ingredients: DishIngredient[];
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

// --- training (exercises & workouts) ---

export type CatalogScope = "all" | "mine" | "favorites";
export type CatalogSort = "new" | "rating" | "name";
export type Category = "strength" | "cardio" | "mobility";
export type Difficulty = "easy" | "medium" | "hard";
export type JointImpact = "low" | "medium" | "high";

export interface Option {
  key: string;
  label: string;
}

export interface TrainingMeta {
  categories: Option[];
  difficulties: Option[];
  jointImpacts: Option[];
  equipment: Option[];
  muscles: Option[];
}

export interface Exercise {
  id: string;
  name: string;
  description: string;
  category: Category;
  difficulty: Difficulty;
  jointImpact: JointImpact;
  equipment: string[];
  muscles: string[];
  imageUrl: string | null;
  videoUrl: string;
  ratingAvg: number;
  ratingCount: number;
  myRating: number | null;
  isFavorite: boolean;
  isMine: boolean;
  isPublic: boolean;
  authorName: string;
  createdAt: string;
}

export interface ExerciseRef {
  id: string;
  name: string;
  category: Category;
  difficulty: Difficulty;
  jointImpact: JointImpact;
  equipment: string[];
  imageUrl: string | null;
  ratingAvg: number;
  ratingCount: number;
}

export interface WorkoutItem {
  id: string;
  exerciseId: string;
  position: number;
  sets: number | null;
  reps: number | null;
  durationSec: number | null;
  restSec: number | null;
  weightKg: number | null;
  note: string;
  exercise: ExerciseRef;
}

export interface Workout {
  id: string;
  name: string;
  description: string;
  difficulty: Difficulty;
  imageUrl: string | null;
  exerciseCount: number;
  ratingAvg: number;
  ratingCount: number;
  myRating: number | null;
  isFavorite: boolean;
  isMine: boolean;
  isPublic: boolean;
  authorName: string;
  createdAt: string;
  items?: WorkoutItem[];
}

export interface CatalogComment {
  id: string;
  authorName: string;
  body: string;
  createdAt: string;
  isMine: boolean;
}

// --- medications / vitamins ---

export type MedStatus = "upcoming" | "active" | "finished";

// --- blood pressure ---

export type PressureCategory = "" | "normal" | "elevated" | "high1" | "high2" | "crisis";

export interface PressureEntry {
  id: string;
  systolic: number;
  diastolic: number;
  pulse: number | null;
  note: string;
  measuredAt: string;
}

export interface PressureAverages {
  systolic: number | null;
  diastolic: number | null;
  pulse: number | null;
}

export interface PressureSummary {
  latest: PressureEntry | null;
  category: PressureCategory;
  averages: PressureAverages;
  series: PressureEntry[];
  count: number;
}

// --- diary ---

export interface DiaryAttachment {
  key: string;
  kind: "image" | "video";
  contentType: string;
  url: string | null;
}

export interface DiaryEntry {
  id: string;
  date: string;
  body: string;
  attachments: DiaryAttachment[];
  createdAt: string;
  updatedAt: string;
}

export interface DiaryDateCount {
  date: string;
  count: number;
}

export interface Medication {
  id: string;
  name: string;
  unit: string;
  dose: number;
  timesPerDay: number;
  startDate: string;
  durationDays: number | null;
  courseDay: number;
  courseTotal: number | null;
  status: MedStatus;
  takenToday: number;
  remainingToday: number;
  weekTaken: number;
  notes: string;
  active: boolean;
  createdAt: string;
}
