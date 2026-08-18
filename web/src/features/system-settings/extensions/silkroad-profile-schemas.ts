/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { z } from 'zod'

export const optionItemSchema = z.object({
  label: z.string().min(1),
  value: z.string().min(1),
  upstream_key: z.string().min(1),
  enabled: z.boolean(),
  sort: z.coerce.number().int(),
})

export const profileFormSchema = z.object({
  id: z.string().min(1),
  label: z.string().min(1),
  exact_models_text: z.string(),
  /** Comma-separated prefixes in the UI; converted to string[] on save. */
  model_prefixes_text: z.string(),
  require_ref_model_suffix: z.boolean(),
  durations: z.array(optionItemSchema),
  aspect_ratios: z.array(optionItemSchema),
})

export type OptionItemForm = z.infer<typeof optionItemSchema>
export type ProfileForm = z.infer<typeof profileFormSchema>

export type ProfileApi = {
  id: string
  label: string
  exact_models: string[]
  model_prefixes: string[]
  require_ref_model_suffix?: boolean
  durations?: OptionItemForm[]
  aspect_ratios?: OptionItemForm[]
}

function asOptionItem(raw: unknown, fallbackSort: number): OptionItemForm {
  const o = (raw ?? {}) as Partial<OptionItemForm>
  return {
    label: String(o.label ?? ''),
    value: String(o.value ?? ''),
    upstream_key: String(o.upstream_key ?? ''),
    enabled: Boolean(o.enabled ?? true),
    sort:
      typeof o.sort === 'number' && Number.isFinite(o.sort)
        ? o.sort
        : fallbackSort,
  }
}

export function emptyOptionItem(sort = 1): OptionItemForm {
  return {
    label: '',
    value: '',
    upstream_key: '',
    enabled: true,
    sort,
  }
}

export function emptyProfile(index = 0): ProfileForm {
  return {
    id: `profile_${index + 1}`,
    label: '',
    exact_models_text: '',
    model_prefixes_text: '',
    require_ref_model_suffix: true,
    durations: [],
    aspect_ratios: [],
  }
}

export function parseProfilesToForm(raw: string | undefined): ProfileForm[] {
  if (!raw || !raw.trim()) return [emptyProfile(0)]
  try {
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed) || parsed.length === 0) {
      return [emptyProfile(0)]
    }
    return parsed.map((item, idx) => {
      const p = (item ?? {}) as Partial<ProfileApi>
      const exactModels = Array.isArray(p.exact_models)
        ? p.exact_models.map(String).filter(Boolean)
        : []
      const prefixes = Array.isArray(p.model_prefixes)
        ? p.model_prefixes.map(String).filter(Boolean)
        : []
      const durations = Array.isArray(p.durations) ? p.durations : []
      const aspects = Array.isArray(p.aspect_ratios) ? p.aspect_ratios : []
      return {
        id: String(p.id ?? `profile_${idx + 1}`),
        label: String(p.label ?? ''),
        exact_models_text: exactModels.join(', '),
        model_prefixes_text: prefixes.join(', '),
        require_ref_model_suffix: p.require_ref_model_suffix !== false,
        durations: durations.map((d, i) => asOptionItem(d, i + 1)),
        aspect_ratios: aspects.map((d, i) => asOptionItem(d, i + 1)),
      }
    })
  } catch {
    return [emptyProfile(0)]
  }
}

export function profilesFormToApi(profiles: ProfileForm[]): ProfileApi[] {
  return profiles.map((p) => ({
    id: p.id.trim(),
    label: p.label.trim(),
    exact_models: p.exact_models_text
      .split(/[,，\n]/)
      .map((s) => s.trim())
      .filter(Boolean),
    model_prefixes: p.model_prefixes_text
      .split(/[,，\n]/)
      .map((s) => s.trim())
      .filter(Boolean),
    ...(p.require_ref_model_suffix ? {} : { require_ref_model_suffix: false }),
    ...(p.durations.length > 0
      ? {
          durations: p.durations.map((d) => ({
            ...d,
            label: d.label.trim(),
            value: d.value.trim(),
            upstream_key: d.upstream_key.trim(),
          })),
        }
      : {}),
    ...(p.aspect_ratios.length > 0
      ? {
          aspect_ratios: p.aspect_ratios.map((d) => ({
            ...d,
            label: d.label.trim(),
            value: d.value.trim(),
            upstream_key: d.upstream_key.trim(),
          })),
        }
      : {}),
  }))
}

export function defaultProfileExists(
  defaultProfileID: string,
  profiles: ProfileForm[]
): boolean {
  return profiles.some((profile) => profile.id.trim() === defaultProfileID)
}
