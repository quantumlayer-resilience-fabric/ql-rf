"use client";

import { useState, useMemo } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Package as PackageIcon, AlertCircle, Search } from "lucide-react";
import type { Package } from "@/lib/api-sbom";

interface SBOMComponentsTableProps {
  components: Package[];
}

type SortField = "name" | "version" | "type" | "license";
type SortDirection = "asc" | "desc";

export function SBOMComponentsTable({ components }: SBOMComponentsTableProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [licenseFilter, setLicenseFilter] = useState<string>("all");
  const [sortField, setSortField] = useState<SortField>("name");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");

  // Extract unique types and licenses for filters
  const uniqueTypes = useMemo(() => {
    const types = new Set(components.map((c) => c.type));
    return Array.from(types).sort();
  }, [components]);

  const uniqueLicenses = useMemo(() => {
    const licenses = new Set(
      components.map((c) => c.license || "Unknown").filter(Boolean)
    );
    return Array.from(licenses).sort();
  }, [components]);

  // Filter and sort components
  const filteredComponents = useMemo(() => {
    let filtered = components;

    // Apply search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (c) =>
          c.name.toLowerCase().includes(query) ||
          c.version.toLowerCase().includes(query) ||
          c.license?.toLowerCase().includes(query)
      );
    }

    // Apply type filter
    if (typeFilter !== "all") {
      filtered = filtered.filter((c) => c.type === typeFilter);
    }

    // Apply license filter
    if (licenseFilter !== "all") {
      filtered = filtered.filter(
        (c) => (c.license || "Unknown") === licenseFilter
      );
    }

    // Apply sorting
    filtered.sort((a, b) => {
      let aVal: string;
      let bVal: string;

      switch (sortField) {
        case "name":
          aVal = a.name;
          bVal = b.name;
          break;
        case "version":
          aVal = a.version;
          bVal = b.version;
          break;
        case "type":
          aVal = a.type;
          bVal = b.type;
          break;
        case "license":
          aVal = a.license || "Unknown";
          bVal = b.license || "Unknown";
          break;
        default:
          aVal = a.name;
          bVal = b.name;
      }

      const comparison = aVal.localeCompare(bVal);
      return sortDirection === "asc" ? comparison : -comparison;
    });

    return filtered;
  }, [components, searchQuery, typeFilter, licenseFilter, sortField, sortDirection]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDirection("asc");
    }
  };

  return (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search components..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <div className="flex gap-2">
          <Select value={typeFilter} onValueChange={setTypeFilter}>
            <SelectTrigger className="w-[140px]">
              <SelectValue placeholder="Type" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Types</SelectItem>
              {uniqueTypes.map((type) => (
                <SelectItem key={type} value={type}>
                  {type}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={licenseFilter} onValueChange={setLicenseFilter}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="License" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Licenses</SelectItem>
              {uniqueLicenses.map((license) => (
                <SelectItem key={license} value={license}>
                  {license}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Results count */}
      <div className="text-sm text-muted-foreground">
        Showing {filteredComponents.length} of {components.length} components
      </div>

      {/* Table */}
      <div className="rounded-lg border">
        <table className="w-full">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="px-4 py-3 text-left">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleSort("name")}
                  className="h-auto p-0 font-medium hover:bg-transparent"
                >
                  Component
                  {sortField === "name" && (
                    <span className="ml-1">{sortDirection === "asc" ? "↑" : "↓"}</span>
                  )}
                </Button>
              </th>
              <th className="px-4 py-3 text-left">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleSort("version")}
                  className="h-auto p-0 font-medium hover:bg-transparent"
                >
                  Version
                  {sortField === "version" && (
                    <span className="ml-1">{sortDirection === "asc" ? "↑" : "↓"}</span>
                  )}
                </Button>
              </th>
              <th className="px-4 py-3 text-left">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleSort("type")}
                  className="h-auto p-0 font-medium hover:bg-transparent"
                >
                  Type
                  {sortField === "type" && (
                    <span className="ml-1">{sortDirection === "asc" ? "↑" : "↓"}</span>
                  )}
                </Button>
              </th>
              <th className="px-4 py-3 text-left">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleSort("license")}
                  className="h-auto p-0 font-medium hover:bg-transparent"
                >
                  License
                  {sortField === "license" && (
                    <span className="ml-1">{sortDirection === "asc" ? "↑" : "↓"}</span>
                  )}
                </Button>
              </th>
              <th className="px-4 py-3 text-left">Location</th>
            </tr>
          </thead>
          <tbody>
            {filteredComponents.length > 0 ? (
              filteredComponents.map((component, index) => (
                <tr
                  key={component.id}
                  className={index !== filteredComponents.length - 1 ? "border-b" : ""}
                >
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <PackageIcon className="h-4 w-4 text-muted-foreground" />
                      <div>
                        <div className="font-medium">{component.name}</div>
                        {component.purl && (
                          <div className="text-xs text-muted-foreground">
                            {component.purl}
                          </div>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <code className="rounded bg-muted px-2 py-0.5 text-xs">
                      {component.version}
                    </code>
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant="outline" className="text-xs">
                      {component.type}
                    </Badge>
                  </td>
                  <td className="px-4 py-3">
                    {component.license ? (
                      <Badge
                        variant="secondary"
                        className="text-xs"
                      >
                        {component.license}
                      </Badge>
                    ) : (
                      <span className="flex items-center gap-1 text-xs text-muted-foreground">
                        <AlertCircle className="h-3 w-3" />
                        Unknown
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {component.location && (
                      <div className="truncate text-xs text-muted-foreground max-w-[200px]">
                        {component.location}
                      </div>
                    )}
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                  No components found matching your filters
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
