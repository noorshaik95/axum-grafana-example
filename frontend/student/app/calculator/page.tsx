'use client';

import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Progress } from '@/components/ui/progress';
import { Calculator, Plus, Trash2, RotateCcw, TrendingUp, TrendingDown } from 'lucide-react';
import { mockGradeCategories } from '@/lib/mock-data-extended';

interface GradeEntry {
  id: string;
  name: string;
  score: number;
  maxScore: number;
  categoryId: string;
}

export default function CalculatorPage() {
  const [categories] = useState(mockGradeCategories);
  const [grades, setGrades] = useState<GradeEntry[]>([
    { id: '1', name: 'Quiz 1', score: 85, maxScore: 100, categoryId: 'cat-1' },
    { id: '2', name: 'Assignment 1', score: 90, maxScore: 100, categoryId: 'cat-2' },
    { id: '3', name: 'Midterm Exam', score: 78, maxScore: 100, categoryId: 'cat-3' },
  ]);

  const [newGrade, setNewGrade] = useState({
    name: '',
    score: '',
    maxScore: '',
    categoryId: categories[0].id,
  });

  const calculateCategoryAverage = (categoryId: string) => {
    const categoryGrades = grades.filter((g) => g.categoryId === categoryId);
    if (categoryGrades.length === 0) return 0;

    const totalEarned = categoryGrades.reduce((sum, g) => sum + g.score, 0);
    const totalPossible = categoryGrades.reduce((sum, g) => sum + g.maxScore, 0);

    return totalPossible > 0 ? (totalEarned / totalPossible) * 100 : 0;
  };

  const calculateOverallGrade = () => {
    let weightedSum = 0;
    let totalWeight = 0;

    categories.forEach((category) => {
      const avg = calculateCategoryAverage(category.id);
      const categoryGrades = grades.filter((g) => g.categoryId === category.id);

      if (categoryGrades.length > 0) {
        weightedSum += avg * (category.weight / 100);
        totalWeight += category.weight;
      }
    });

    return totalWeight > 0 ? (weightedSum / totalWeight) * 100 : 0;
  };

  const calculateProjectedGrade = (categoryId: string, targetScore: number) => {
    const category = categories.find((c) => c.id === categoryId);
    if (!category) return calculateOverallGrade();

    // Simulate adding a perfect score to this category
    const tempGrades = [
      ...grades,
      {
        id: 'temp',
        name: 'Projected',
        score: targetScore,
        maxScore: 100,
        categoryId,
      },
    ];

    let weightedSum = 0;
    let totalWeight = 0;

    categories.forEach((cat) => {
      const categoryGradesList = tempGrades.filter((g) => g.categoryId === cat.id);
      if (categoryGradesList.length === 0) return;

      const totalEarned = categoryGradesList.reduce((sum, g) => sum + g.score, 0);
      const totalPossible = categoryGradesList.reduce((sum, g) => sum + g.maxScore, 0);
      const avg = totalPossible > 0 ? (totalEarned / totalPossible) * 100 : 0;

      weightedSum += avg * (cat.weight / 100);
      totalWeight += cat.weight;
    });

    return totalWeight > 0 ? (weightedSum / totalWeight) * 100 : 0;
  };

  const getGradeColor = (percentage: number) => {
    if (percentage >= 90) return 'text-green-600';
    if (percentage >= 80) return 'text-blue-600';
    if (percentage >= 70) return 'text-yellow-600';
    if (percentage >= 60) return 'text-orange-600';
    return 'text-red-600';
  };

  const getGradeLetter = (percentage: number) => {
    if (percentage >= 90) return 'A';
    if (percentage >= 80) return 'B';
    if (percentage >= 70) return 'C';
    if (percentage >= 60) return 'D';
    return 'F';
  };

  const addGrade = () => {
    if (!newGrade.name || !newGrade.score || !newGrade.maxScore) return;

    const grade: GradeEntry = {
      id: Date.now().toString(),
      name: newGrade.name,
      score: parseFloat(newGrade.score),
      maxScore: parseFloat(newGrade.maxScore),
      categoryId: newGrade.categoryId,
    };

    setGrades([...grades, grade]);
    setNewGrade({
      name: '',
      score: '',
      maxScore: '',
      categoryId: categories[0].id,
    });
  };

  const removeGrade = (id: string) => {
    setGrades(grades.filter((g) => g.id !== id));
  };

  const resetCalculator = () => {
    setGrades([]);
  };

  const overallGrade = calculateOverallGrade();
  const currentLetter = getGradeLetter(overallGrade);

  return (
    <div className="space-y-6 max-w-6xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Grade Calculator</h1>
          <p className="text-muted-foreground">
            Calculate your grades and explore what-if scenarios
          </p>
        </div>
        <Button variant="outline" onClick={resetCalculator}>
          <RotateCcw className="mr-2 h-4 w-4" />
          Reset
        </Button>
      </div>

      {/* Overall Grade */}
      <Card className="border-2 border-primary">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg">Overall Grade</CardTitle>
              <CardDescription>Based on all entered grades</CardDescription>
            </div>
            <Badge className="text-lg px-4 py-2" variant="default">
              {currentLetter}
            </Badge>
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-muted-foreground">Current Grade</span>
                <span className={`text-4xl font-bold ${getGradeColor(overallGrade)}`}>
                  {overallGrade.toFixed(2)}%
                </span>
              </div>
              <Progress value={overallGrade} className="h-3" />
            </div>

            <div className="grid grid-cols-4 gap-4 pt-4 border-t">
              <div className="text-center">
                <p className="text-2xl font-bold text-green-600">A</p>
                <p className="text-xs text-muted-foreground">90-100%</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold text-blue-600">B</p>
                <p className="text-xs text-muted-foreground">80-89%</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold text-yellow-600">C</p>
                <p className="text-xs text-muted-foreground">70-79%</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold text-orange-600">D</p>
                <p className="text-xs text-muted-foreground">60-69%</p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-6 md:grid-cols-2">
        {/* Category Breakdown */}
        <Card>
          <CardHeader>
            <CardTitle>Category Breakdown</CardTitle>
            <CardDescription>Grade distribution by category</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {categories.map((category) => {
              const avg = calculateCategoryAverage(category.id);
              const categoryGrades = grades.filter((g) => g.categoryId === category.id);

              return (
                <div key={category.id} className="space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{category.name}</span>
                      <Badge variant="secondary" className="text-xs">
                        {category.weight}%
                      </Badge>
                    </div>
                    <div className="text-right">
                      <span className={`font-bold ${getGradeColor(avg)}`}>
                        {avg > 0 ? avg.toFixed(1) : '--'}%
                      </span>
                      <p className="text-xs text-muted-foreground">
                        {categoryGrades.length} item{categoryGrades.length !== 1 ? 's' : ''}
                      </p>
                    </div>
                  </div>
                  <Progress value={avg} className="h-2" />
                </div>
              );
            })}
          </CardContent>
        </Card>

        {/* What-If Scenarios */}
        <Card>
          <CardHeader>
            <CardTitle>What-If Scenarios</CardTitle>
            <CardDescription>See how different scores affect your grade</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {categories.map((category) => {
              const currentGrade = overallGrade;
              const withPerfect = calculateProjectedGrade(category.id, 100);
              const withZero = calculateProjectedGrade(category.id, 0);
              const difference = withPerfect - currentGrade;

              return (
                <div key={category.id} className="p-4 border rounded-lg space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="font-medium">{category.name}</span>
                    <Badge variant="outline">{category.weight}%</Badge>
                  </div>

                  <div className="space-y-2 text-sm">
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">If next score is 100%:</span>
                      <div className="flex items-center gap-2">
                        <span className={`font-bold ${getGradeColor(withPerfect)}`}>
                          {withPerfect.toFixed(1)}%
                        </span>
                        {difference > 0 && (
                          <Badge variant="outline" className="text-green-600 text-xs">
                            <TrendingUp className="h-3 w-3 mr-1" />+{difference.toFixed(1)}
                          </Badge>
                        )}
                      </div>
                    </div>

                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">If next score is 0%:</span>
                      <div className="flex items-center gap-2">
                        <span className={`font-bold ${getGradeColor(withZero)}`}>
                          {withZero.toFixed(1)}%
                        </span>
                        <Badge variant="outline" className="text-red-600 text-xs">
                          <TrendingDown className="h-3 w-3 mr-1" />
                          {(withZero - currentGrade).toFixed(1)}
                        </Badge>
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </CardContent>
        </Card>
      </div>

      {/* Add New Grade */}
      <Card>
        <CardHeader>
          <CardTitle>Add Grade</CardTitle>
          <CardDescription>Enter a new grade to update your calculations</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-5">
            <div className="md:col-span-2">
              <label className="text-sm font-medium mb-2 block">Assignment Name</label>
              <Input
                placeholder="e.g., Quiz 1"
                value={newGrade.name}
                onChange={(e) => setNewGrade({ ...newGrade, name: e.target.value })}
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-2 block">Score</label>
              <Input
                type="number"
                placeholder="85"
                value={newGrade.score}
                onChange={(e) => setNewGrade({ ...newGrade, score: e.target.value })}
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-2 block">Max Score</label>
              <Input
                type="number"
                placeholder="100"
                value={newGrade.maxScore}
                onChange={(e) => setNewGrade({ ...newGrade, maxScore: e.target.value })}
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-2 block">Category</label>
              <select
                className="w-full h-10 px-3 border rounded-md bg-background"
                value={newGrade.categoryId}
                onChange={(e) => setNewGrade({ ...newGrade, categoryId: e.target.value })}
              >
                {categories.map((cat) => (
                  <option key={cat.id} value={cat.id}>
                    {cat.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <Button className="mt-4" onClick={addGrade}>
            <Plus className="mr-2 h-4 w-4" />
            Add Grade
          </Button>
        </CardContent>
      </Card>

      {/* Current Grades */}
      <Card>
        <CardHeader>
          <CardTitle>Entered Grades</CardTitle>
          <CardDescription>
            {grades.length} grade{grades.length !== 1 ? 's' : ''} entered
          </CardDescription>
        </CardHeader>
        <CardContent>
          {grades.length > 0 ? (
            <div className="space-y-2">
              {grades.map((grade) => {
                const category = categories.find((c) => c.id === grade.categoryId);
                const percentage = (grade.score / grade.maxScore) * 100;

                return (
                  <div
                    key={grade.id}
                    className="flex items-center justify-between p-4 border rounded-lg"
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <p className="font-medium">{grade.name}</p>
                        <Badge variant="secondary" className="text-xs">
                          {category?.name}
                        </Badge>
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {grade.score} / {grade.maxScore} points
                      </p>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="text-right">
                        <p className={`text-2xl font-bold ${getGradeColor(percentage)}`}>
                          {percentage.toFixed(1)}%
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {getGradeLetter(percentage)}
                        </p>
                      </div>
                      <Button variant="ghost" size="icon" onClick={() => removeGrade(grade.id)}>
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="text-center py-12">
              <Calculator className="mx-auto h-12 w-12 text-muted-foreground/50 mb-4" />
              <p className="text-muted-foreground">
                No grades entered yet. Add grades above to start calculating.
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
