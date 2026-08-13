/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

export function normalizeProviderGroups(value: string): string[] {
  return [
    ...new Set(
      value
        .split(/[,，\n]/)
        .map((group) => group.trim())
        .filter(Boolean)
    ),
  ]
}

export function parseProviderGroups(raw: string): string {
  try {
    const groups = JSON.parse(raw) as unknown
    if (!Array.isArray(groups)) return ''
    return [
      ...new Set(
        groups
          .filter((group): group is string => typeof group === 'string')
          .map((group) => group.trim())
          .filter(Boolean)
      ),
    ].join(', ')
  } catch {
    return ''
  }
}
