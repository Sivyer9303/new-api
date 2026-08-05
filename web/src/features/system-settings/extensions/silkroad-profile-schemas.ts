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

export const upstreamSetSchema = z.object({
  upstream_key: z.string().min(1),
  value: z.string().optional().default(''),
  from: z.string().optional().default(''),
})

export const generationTypeSchema = z.object({
  label: z.string().min(1),
  value: z.string().min(1),
  enabled: z.boolean(),
  sort: z.coerce.number().int(),
  require_ref_model: z.boolean(),
  upstream_sets: z.array(upstreamSetSchema),
  media_requirements: z.object({
    images_min: z.coerce.number().int().min(0),
    images_max: z.coerce.number().int().min(0),
  }),
})

export const profileFormSchema = z.object({
  id: z.string().min(1),
  label: z.string().min(1),
  /** Comma-separated prefixes in the UI; converted to string[] on save. */
  model_prefixes_text: z.string().min(1),
  durations: z.array(optionItemSchema).min(1),
  aspect_ratios: z.array(optionItemSchema).min(1),
  generation_types: z.array(generationTypeSchema).min(1),
  extra_options: z.array(optionItemSchema),
})

export type OptionItemForm = z.infer<typeof optionItemSchema>
export type UpstreamSetForm = z.infer<typeof upstreamSetSchema>
export type GenerationTypeForm = z.infer<typeof generationTypeSchema>
export type ProfileForm = z.infer<typeof profileFormSchema>

export type ProfileApi = {
  id: string
  label: string
  model_prefixes: string[]
  durations: OptionItemForm[]
  aspect_ratios: OptionItemForm[]
  generation_types: Array<{
    label: string
    value: string
    enabled: boolean
    sort: number
    require_ref_model: boolean
    upstream_sets: Array<{
      upstream_key: string
      value?: string
      from?: string
    }>
    media_requirements: {
      images_min: number
      images_max: number
    }
  }>
  extra_options: OptionItemForm[]
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

function asGenerationType(
  raw: unknown,
  fallbackSort: number
): GenerationTypeForm {
  const o = (raw ?? {}) as Partial<GenerationTypeForm> & {
    media_requirements?: Partial<GenerationTypeForm['media_requirements']>
    upstream_sets?: unknown[]
  }
  const sets = Array.isArray(o.upstream_sets) ? o.upstream_sets : []
  return {
    label: String(o.label ?? ''),
    value: String(o.value ?? ''),
    enabled: Boolean(o.enabled ?? true),
    sort:
      typeof o.sort === 'number' && Number.isFinite(o.sort)
        ? o.sort
        : fallbackSort,
    require_ref_model: Boolean(o.require_ref_model),
    upstream_sets: sets.map((s) => {
      const item = (s ?? {}) as Partial<UpstreamSetForm>
      return {
        upstream_key: String(item.upstream_key ?? ''),
        value: String(item.value ?? ''),
        from: String(item.from ?? ''),
      }
    }),
    media_requirements: {
      images_min:
        typeof o.media_requirements?.images_min === 'number'
          ? o.media_requirements.images_min
          : 0,
      images_max:
        typeof o.media_requirements?.images_max === 'number'
          ? o.media_requirements.images_max
          : 0,
    },
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

export function emptyGenerationType(sort = 1): GenerationTypeForm {
  return {
    label: '',
    value: '',
    enabled: true,
    sort,
    require_ref_model: false,
    upstream_sets: [],
    media_requirements: { images_min: 0, images_max: 0 },
  }
}

export function emptyProfile(index = 0): ProfileForm {
  return {
    id: `profile_${index + 1}`,
    label: '',
    model_prefixes_text: '',
    durations: [emptyOptionItem(1)],
    aspect_ratios: [emptyOptionItem(1)],
    generation_types: [emptyGenerationType(1)],
    extra_options: [],
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
      const prefixes = Array.isArray(p.model_prefixes)
        ? p.model_prefixes.map(String).filter(Boolean)
        : []
      const durations = Array.isArray(p.durations) ? p.durations : []
      const aspects = Array.isArray(p.aspect_ratios) ? p.aspect_ratios : []
      const gens = Array.isArray(p.generation_types) ? p.generation_types : []
      const extras = Array.isArray(p.extra_options) ? p.extra_options : []
      return {
        id: String(p.id ?? `profile_${idx + 1}`),
        label: String(p.label ?? ''),
        model_prefixes_text: prefixes.join(', '),
        durations:
          durations.length > 0
            ? durations.map((d, i) => asOptionItem(d, i + 1))
            : [emptyOptionItem(1)],
        aspect_ratios:
          aspects.length > 0
            ? aspects.map((d, i) => asOptionItem(d, i + 1))
            : [emptyOptionItem(1)],
        generation_types:
          gens.length > 0
            ? gens.map((d, i) => asGenerationType(d, i + 1))
            : [emptyGenerationType(1)],
        extra_options: extras.map((d, i) => asOptionItem(d, i + 1)),
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
    model_prefixes: p.model_prefixes_text
      .split(/[,，\n]/)
      .map((s) => s.trim())
      .filter(Boolean),
    durations: p.durations.map((d) => ({
      ...d,
      label: d.label.trim(),
      value: d.value.trim(),
      upstream_key: d.upstream_key.trim(),
    })),
    aspect_ratios: p.aspect_ratios.map((d) => ({
      ...d,
      label: d.label.trim(),
      value: d.value.trim(),
      upstream_key: d.upstream_key.trim(),
    })),
    generation_types: p.generation_types.map((g) => ({
      label: g.label.trim(),
      value: g.value.trim(),
      enabled: g.enabled,
      sort: g.sort,
      require_ref_model: g.require_ref_model,
      upstream_sets: g.upstream_sets
        .map((s) => ({
          upstream_key: s.upstream_key.trim(),
          ...(s.value?.trim() ? { value: s.value.trim() } : {}),
          ...(s.from?.trim() ? { from: s.from.trim() } : {}),
        }))
        .filter((s) => s.upstream_key),
      media_requirements: {
        images_min: g.media_requirements.images_min,
        images_max: g.media_requirements.images_max,
      },
    })),
    extra_options: p.extra_options.map((d) => ({
      ...d,
      label: d.label.trim(),
      value: d.value.trim(),
      upstream_key: d.upstream_key.trim(),
    })),
  }))
}
