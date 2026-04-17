'use client';

import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Progress } from '@/components/ui/progress';
import {
  File,
  Folder,
  Upload,
  Download,
  Share2,
  Trash2,
  Search,
  Grid3x3,
  List,
  FileText,
  FileCode,
  Image as ImageIcon,
  Film,
  Archive,
  MoreVertical,
  Plus,
  HardDrive,
  ChevronRight,
} from 'lucide-react';
import { mockFiles, mockFolders } from '@/lib/mock-data-extended';
import { formatFileSize, formatRelativeTime } from '@/lib/utils';

export default function FilesPage() {
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('list');
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
  const [dragActive, setDragActive] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  // Calculate storage usage
  const totalStorage = 5 * 1024 * 1024 * 1024; // 5GB in bytes
  const usedStorage = mockFiles.reduce((sum, file) => sum + file.size, 0);
  const storagePercentage = (usedStorage / totalStorage) * 100;

  // Filter files by folder and search
  const filteredFiles = mockFiles.filter((file) => {
    const matchesFolder = selectedFolder ? file.folderId === selectedFolder : true;
    const matchesSearch = file.name.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesFolder && matchesSearch;
  });

  const getFileIcon = (type: string) => {
    switch (type) {
      case 'pdf':
      case 'document':
        return FileText;
      case 'code':
        return FileCode;
      case 'image':
        return ImageIcon;
      case 'video':
        return Film;
      case 'archive':
        return Archive;
      default:
        return File;
    }
  };

  const getFileColor = (type: string) => {
    switch (type) {
      case 'pdf':
        return 'text-red-500';
      case 'document':
        return 'text-blue-500';
      case 'code':
        return 'text-green-500';
      case 'image':
        return 'text-purple-500';
      case 'video':
        return 'text-orange-500';
      case 'archive':
        return 'text-gray-500';
      default:
        return 'text-gray-400';
    }
  };

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      // Handle file upload
      console.log('Files dropped:', e.dataTransfer.files);
    }
  };

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      // Handle file upload
      console.log('Files selected:', e.target.files);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Files</h1>
          <p className="text-muted-foreground">Manage your course files and documents</p>
        </div>
        <div className="flex items-center gap-2">
          <label htmlFor="file-upload">
            <Button asChild>
              <span className="cursor-pointer">
                <Upload className="mr-2 h-4 w-4" />
                Upload Files
              </span>
            </Button>
          </label>
          <input
            id="file-upload"
            type="file"
            multiple
            className="hidden"
            onChange={handleFileUpload}
          />
          <Button variant="outline">
            <Plus className="mr-2 h-4 w-4" />
            New Folder
          </Button>
        </div>
      </div>

      {/* Storage Overview */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <HardDrive className="h-5 w-5 text-primary" />
              <CardTitle>Storage</CardTitle>
            </div>
            <CardDescription>
              {formatFileSize(usedStorage)} of {formatFileSize(totalStorage)} used
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent>
          <Progress value={storagePercentage} className="h-2" />
          <div className="mt-4 grid grid-cols-4 gap-4 text-sm">
            <div className="space-y-1">
              <p className="text-muted-foreground">Documents</p>
              <p className="font-semibold">
                {formatFileSize(
                  mockFiles
                    .filter((f) => f.type === 'document' || f.type === 'pdf')
                    .reduce((sum, f) => sum + f.size, 0)
                )}
              </p>
            </div>
            <div className="space-y-1">
              <p className="text-muted-foreground">Images</p>
              <p className="font-semibold">
                {formatFileSize(
                  mockFiles.filter((f) => f.type === 'image').reduce((sum, f) => sum + f.size, 0)
                )}
              </p>
            </div>
            <div className="space-y-1">
              <p className="text-muted-foreground">Videos</p>
              <p className="font-semibold">
                {formatFileSize(
                  mockFiles.filter((f) => f.type === 'video').reduce((sum, f) => sum + f.size, 0)
                )}
              </p>
            </div>
            <div className="space-y-1">
              <p className="text-muted-foreground">Other</p>
              <p className="font-semibold">
                {formatFileSize(
                  mockFiles
                    .filter((f) => f.type === 'code' || f.type === 'archive')
                    .reduce((sum, f) => sum + f.size, 0)
                )}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-6 md:grid-cols-4">
        {/* Folder Tree */}
        <Card className="md:col-span-1">
          <CardHeader>
            <CardTitle className="text-base">Folders</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1">
            <button
              onClick={() => setSelectedFolder(null)}
              className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
                selectedFolder === null ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'
              }`}
            >
              <Folder className="h-4 w-4" />
              <span>All Files</span>
            </button>
            {mockFolders.map((folder) => {
              const fileCount = mockFiles.filter((f) => f.folderId === folder.id).length;
              return (
                <button
                  key={folder.id}
                  onClick={() => setSelectedFolder(folder.id)}
                  className={`w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm transition-colors ${
                    selectedFolder === folder.id
                      ? 'bg-primary text-primary-foreground'
                      : 'hover:bg-accent'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <Folder className="h-4 w-4" />
                    <span className="truncate">{folder.name}</span>
                  </div>
                  <Badge variant="secondary" className="text-xs">
                    {fileCount}
                  </Badge>
                </button>
              );
            })}
          </CardContent>
        </Card>

        {/* File List */}
        <div className="md:col-span-3 space-y-4">
          {/* Toolbar */}
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between gap-4">
                <div className="flex-1 relative">
                  <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    placeholder="Search files..."
                    className="pl-10"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                  />
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant={viewMode === 'list' ? 'default' : 'outline'}
                    size="icon"
                    onClick={() => setViewMode('list')}
                  >
                    <List className="h-4 w-4" />
                  </Button>
                  <Button
                    variant={viewMode === 'grid' ? 'default' : 'outline'}
                    size="icon"
                    onClick={() => setViewMode('grid')}
                  >
                    <Grid3x3 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Breadcrumb */}
          {selectedFolder && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <button
                onClick={() => setSelectedFolder(null)}
                className="hover:text-foreground transition-colors"
              >
                All Files
              </button>
              <ChevronRight className="h-4 w-4" />
              <span className="text-foreground font-medium">
                {mockFolders.find((f) => f.id === selectedFolder)?.name}
              </span>
            </div>
          )}

          {/* Upload Area */}
          <div
            className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
              dragActive ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/50'
            }`}
            onDragEnter={handleDrag}
            onDragLeave={handleDrag}
            onDragOver={handleDrag}
            onDrop={handleDrop}
          >
            <Upload className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-lg font-medium mb-2">
              {dragActive ? 'Drop files here' : 'Drag and drop files here'}
            </p>
            <p className="text-sm text-muted-foreground mb-4">or click the upload button above</p>
            <p className="text-xs text-muted-foreground">
              Supports: PDF, DOC, DOCX, XLS, XLSX, PPT, PPTX, images, videos, and more
            </p>
          </div>

          {/* Files */}
          {viewMode === 'list' ? (
            <Card>
              <CardContent className="p-0">
                <div className="divide-y">
                  {filteredFiles.length > 0 ? (
                    filteredFiles.map((file) => {
                      const Icon = getFileIcon(file.type);
                      const iconColor = getFileColor(file.type);
                      const folder = mockFolders.find((f) => f.id === file.folderId);

                      return (
                        <div
                          key={file.id}
                          className="flex items-center justify-between p-4 hover:bg-accent transition-colors"
                        >
                          <div className="flex items-center gap-4 flex-1 min-w-0">
                            <Icon className={`h-8 w-8 flex-shrink-0 ${iconColor}`} />
                            <div className="flex-1 min-w-0">
                              <p className="font-medium truncate">{file.name}</p>
                              <div className="flex items-center gap-4 text-sm text-muted-foreground">
                                <span>{formatFileSize(file.size)}</span>
                                <span>•</span>
                                <span>{formatRelativeTime(file.uploadedAt)}</span>
                                {folder && !selectedFolder && (
                                  <>
                                    <span>•</span>
                                    <span className="flex items-center gap-1">
                                      <Folder className="h-3 w-3" />
                                      {folder.name}
                                    </span>
                                  </>
                                )}
                              </div>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            <Button variant="ghost" size="icon">
                              <Download className="h-4 w-4" />
                            </Button>
                            <Button variant="ghost" size="icon">
                              <Share2 className="h-4 w-4" />
                            </Button>
                            <Button variant="ghost" size="icon">
                              <Trash2 className="h-4 w-4 text-destructive" />
                            </Button>
                            <Button variant="ghost" size="icon">
                              <MoreVertical className="h-4 w-4" />
                            </Button>
                          </div>
                        </div>
                      );
                    })
                  ) : (
                    <div className="p-12 text-center">
                      <File className="mx-auto h-12 w-12 text-muted-foreground/50 mb-4" />
                      <p className="text-muted-foreground">
                        {searchQuery ? 'No files match your search' : 'No files in this folder'}
                      </p>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
              {filteredFiles.length > 0 ? (
                filteredFiles.map((file) => {
                  const Icon = getFileIcon(file.type);
                  const iconColor = getFileColor(file.type);

                  return (
                    <Card key={file.id} className="hover:shadow-md transition-all">
                      <CardContent className="p-4">
                        <div className="aspect-square rounded-lg bg-accent flex items-center justify-center mb-3">
                          <Icon className={`h-16 w-16 ${iconColor}`} />
                        </div>
                        <p className="font-medium truncate mb-1">{file.name}</p>
                        <p className="text-sm text-muted-foreground mb-3">
                          {formatFileSize(file.size)}
                        </p>
                        <div className="flex gap-1">
                          <Button variant="outline" size="sm" className="flex-1">
                            <Download className="h-3 w-3 mr-1" />
                            Download
                          </Button>
                          <Button variant="outline" size="icon" className="h-8 w-8">
                            <Share2 className="h-3 w-3" />
                          </Button>
                        </div>
                      </CardContent>
                    </Card>
                  );
                })
              ) : (
                <div className="col-span-full p-12 text-center">
                  <File className="mx-auto h-12 w-12 text-muted-foreground/50 mb-4" />
                  <p className="text-muted-foreground">
                    {searchQuery ? 'No files match your search' : 'No files in this folder'}
                  </p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
