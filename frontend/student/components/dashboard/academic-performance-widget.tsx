'use client';

import { GraduationCap, BookOpen, Clock, TrendingUp, Award } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card';

export function AcademicPerformanceWidget() {
  return (
    <Card className="glass-card">
      <CardHeader>
        <CardTitle className="text-foreground">Academic Performance</CardTitle>
        <CardDescription className="text-muted-foreground">
          Your overall academic statistics
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4">
            <div className="rounded-xl p-4 glass-card">
              <div className="flex items-center gap-2 mb-2">
                <GraduationCap className="w-5 h-5 text-muted-foreground" />
                <span className="text-sm font-medium text-muted-foreground">GPA</span>
              </div>
              <p className="text-3xl font-bold text-foreground">3.85</p>
              <p className="text-xs text-muted-foreground mt-1">Out of 4.0</p>
            </div>

            <div className="rounded-xl p-4 glass-card">
              <div className="flex items-center gap-2 mb-2">
                <TrendingUp className="w-5 h-5 text-muted-foreground" />
                <span className="text-sm font-medium text-muted-foreground">Average</span>
              </div>
              <p className="text-3xl font-bold text-foreground">88%</p>
              <p className="text-xs text-muted-foreground mt-1">All courses</p>
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between glass-card rounded-lg p-3">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-slate-200 dark:bg-slate-700">
                  <Award className="w-5 h-5 text-foreground" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">Total Credits</p>
                  <p className="text-xs text-muted-foreground">Completed</p>
                </div>
              </div>
              <p className="text-2xl font-bold text-foreground">45</p>
            </div>

            {/* Enrolled Courses */}
            <div className="flex items-center justify-between glass-card rounded-lg p-3">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-slate-200 dark:bg-slate-700">
                  <BookOpen className="w-5 h-5 text-foreground" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">Enrolled Courses</p>
                  <p className="text-xs text-muted-foreground">This semester</p>
                </div>
              </div>
              <p className="text-2xl font-bold text-foreground">4</p>
            </div>

            {/* Study Time */}
            <div className="flex items-center justify-between glass-card rounded-lg p-3">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-slate-200 dark:bg-slate-700">
                  <Clock className="w-5 h-5 text-foreground" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">Study Time</p>
                  <p className="text-xs text-muted-foreground">This week</p>
                </div>
              </div>
              <p className="text-2xl font-bold text-foreground">24h</p>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
