import React from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  Controller,
  useForm,
  type Control,
  type FieldPath,
} from 'react-hook-form';
import { toast } from 'sonner';
import { z } from 'zod';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLegend,
  FieldLabel,
  FieldSeparator,
  FieldSet,
} from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from '@/components/ui/input-group';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';

import {
  ModelProtocols,
  ModelSchema,
  type ModelFormData,
} from '@/validation/ModelFormSchema';
import {
  CornerDownLeft,
  Server,
  SquareStack,
  X,
  Route,
  Unplug,
  ShieldCogCorner,
  SquareActivity,
  Tag,
} from 'lucide-react';

const FORM_ID = 'model-form';

// The schema's defaults + superRefine make the parser's input type (defaulted
// fields optional) differ from its output type (ModelFormData). RHF is driven by
// the input type; handleSubmit yields the transformed output type.
type FormInput = z.input<typeof ModelSchema>;
type FormControl = Control<FormInput, unknown, ModelFormData>;
type FieldProps = {
  control: FormControl;
  name: FieldPath<FormInput>;
  label: string;
  description?: string;
};

const defaultValues: FormInput = {
  name: '',
  protocol: 'h1',
  baseUrl: '',
  enabled: false,
  modelsServed: [],
  weight: 100,
  maxConcurrent: 1,
  kvCacheAwareRouting: false,
  metricsUrl: undefined,
  scrapeInterval: 15,
  maxIdleConnectionsPerHost: 32,
  idleConnectionTimeout: 90,
  dialTimeout: 5,
  streamStallTimeout: 30,
  responseHeaderTimeout: 30,
  failureThreshold: 5,
  rollingWindow: 10,
  openBase: 1,
  openMax: 30,
  backoffFactor: 2,
  halfOpenProbes: 2,
  halfOpenSuccesses: 2,
  healthCheckPath: '/health',
  healthInterval: 10,
  verifyTlsCert: false,
  description: undefined,
  labels: [],
};

interface ModelFormProps {
  close: () => void;
}

export function ModelForm(props: ModelFormProps) {
  const form = useForm<FormInput, unknown, ModelFormData>({
    resolver: zodResolver(ModelSchema),
    defaultValues,
  });

  const { control } = form;

  const onSubmit = (data: ModelFormData) => {
    console.log(data);
    toast.success(`Backend "${data.name}" saved.`, { position: "top-center" });
  };

  const onInvalid = () => {
    toast.error('Please fix the highlighted fields before saving.', { position: "top-center" });
  };

  return (
    <Card className="w-full max-w-3xl">
      <CardHeader className="flex border-b flex-row justify-between">
        <div>
          <CardTitle>Add Backend</CardTitle>
          <CardDescription>Configure backend parameters</CardDescription>
        </div>
        <div className="flex gap-4">
          <Button
            className="cursor-pointer"
            type="button"
            variant="destructive"
            size="lg"
            onClick={() => {
              props.close();
              form.reset(defaultValues);
            }}
          >
            Cancel
          </Button>
          <Button
            className="cursor-pointer"
            type="submit"
            form={FORM_ID}
            size="lg"
          >
            Save backend
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <form id={FORM_ID} onSubmit={form.handleSubmit(onSubmit, onInvalid)}>
          <FieldGroup>
            <FieldSet>
              <FieldLegend className="flex flex-row gap-1 items-center">
                <Server size={16} />
                <p>Endpoint</p>
              </FieldLegend>
              <FieldGroup>
                <div className="grid gap-4 sm:grid-cols-2">
                  <TextField
                    control={control}
                    name="name"
                    label="Name"
                    placeholder="vllm-a100-3"
                    description="Used as the backend label in metrics, logs, and traces."
                  />
                  <Controller
                    control={control}
                    name="protocol"
                    render={({ field, fieldState }) => (
                      <Field data-invalid={fieldState.invalid}>
                        <FieldLabel htmlFor="protocol">Protocol</FieldLabel>
                        <Select
                          value={field.value}
                          onValueChange={field.onChange}
                        >
                          <SelectTrigger
                            id="protocol"
                            className="w-full"
                            aria-invalid={fieldState.invalid}
                          >
                            <SelectValue placeholder="Select a protocol" />
                          </SelectTrigger>
                          <SelectContent>
                            {ModelProtocols.map((protocol) => (
                              <SelectItem
                                key={protocol.value}
                                value={protocol.value}
                              >
                                {protocol.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <FieldDescription>
                          vLLM/uvicorn speak h1; use h2/h2c only behind a proxy.
                        </FieldDescription>
                        <FieldError
                          errors={
                            fieldState.error ? [fieldState.error] : undefined
                          }
                        />
                      </Field>
                    )}
                  />
                </div>
                <TextField
                  control={control}
                  name="baseUrl"
                  label="Base URL"
                  placeholder="http://vllm-3.vllm.svc:8000"
                  description="Scheme, host, and port of the backend server."
                />
                <SwitchField
                  control={control}
                  name="enabled"
                  label="Enabled"
                  description="Off removes it from rotation without deleting."
                />
              </FieldGroup>
            </FieldSet>

            <FieldSeparator />

            <FieldSet>
              <FieldLegend className="flex flex-row gap-1 items-center">
                <SquareStack size={16} />
                <p>Models served</p>
              </FieldLegend>
              <TagField
                control={control}
                name="modelsServed"
                label="Models served"
                placeholder="Type a model id and press Enter"
                description="Requests for these model ids are eligible to route here (capability filter)"
              />
            </FieldSet>

            <FieldSeparator />

            <FieldSet>
              <FieldLegend className="flex flex-row gap-1 items-center">
                <Route size={16} />
                <p>Routing &amp; capacity</p>
              </FieldLegend>
              <FieldGroup>
                <div className="grid gap-4 sm:grid-cols-2">
                  <NumberField
                    control={control}
                    name="weight"
                    label="Weight"
                    description="Relative capacity hint for P2C scoring."
                  />
                  <NumberField
                    control={control}
                    name="maxConcurrent"
                    label="Max concurrent"
                    description="Per-backend in-flight ceiling. (0 = unlimited)"
                  />
                </div>
                <SwitchField
                  control={control}
                  name="kvCacheAwareRouting"
                  label="KV-cache-aware routing"
                  description="Scrape vLLM /metrics to route by cache pressure."
                />
                <div className="grid gap-4 sm:grid-cols-2">
                  <TextField
                    control={control}
                    name="metricsUrl"
                    label="Metrics URL"
                    optional
                    placeholder="http://vllm-3.vllm.svc:8000/metrics"
                    description="Required when KV-cache-aware routing is on."
                  />
                  <NumberField
                    control={control}
                    name="scrapeInterval"
                    label="Scrape interval (seconds)"
                    description="How often to scrape the metrics endpoint."
                  />
                </div>
              </FieldGroup>
            </FieldSet>

            <FieldSeparator />

            <FieldSet>
              <FieldLegend className="flex flex-row gap-1 items-center">
                <Unplug />
                <p>Connection pool &amp; timeouts</p>
              </FieldLegend>
              <FieldGroup>
                <div className="grid gap-4 sm:grid-cols-5 sm:grid-rows-[auto_auto_auto] sm:gap-x-4 sm:gap-y-0">
                  <NumberField
                    aligned
                    control={control}
                    name="maxIdleConnectionsPerHost"
                    label="Max idle connections / host"
                    description="Idle keep-alive pool size per host."
                  />
                  <NumberField
                    aligned
                    control={control}
                    name="idleConnectionTimeout"
                    label="Idle conn timeout (seconds)"
                    description="How long an idle connection is kept open."
                  />
                  <NumberField
                    aligned
                    control={control}
                    name="dialTimeout"
                    label="Dial timeout (seconds)"
                    description="Max time to establish a new TCP connection."
                  />
                  <NumberField
                    aligned
                    control={control}
                    name="streamStallTimeout"
                    label="Stream-stall timeout (seconds)"
                    description="Abort a stream if no bytes arrive for this long."
                  />
                  <NumberField
                    aligned
                    control={control}
                    name="responseHeaderTimeout"
                    label="Response-header timeout (seconds)"
                    description="Max wait for response headers after sending."
                  />
                </div>
              </FieldGroup>
            </FieldSet>

            <FieldSeparator />

            <FieldSet>
              <FieldLegend className="flex flex-row gap-1 items-center">
                <ShieldCogCorner size={16} />
                <p>Circuit breaker</p>
              </FieldLegend>
              <FieldGroup>
                <div className="grid gap-4 sm:grid-cols-7 sm:grid-rows-[auto_auto_auto] sm:gap-x-4 sm:gap-y-0">
                  <NumberField
                    aligned
                    control={control}
                    name="failureThreshold"
                    label="Failure threshold"
                    description="Failures in the window before the breaker opens."
                  />
                  <NumberField
                    aligned
                    control={control}
                    name="rollingWindow"
                    label="Rolling window (seconds)"
                    description="Window over which failures are counted."
                  />
                  <NumberField
                    aligned
                    control={control}
                    name="openBase"
                    label="Open base (seconds)"
                    description="Initial cool-down before half-open."
                  />
                  <NumberField
                    aligned
                    control={control}
                    name="openMax"
                    label="Open max (seconds)"
                    description="Upper bound on the cool-down."
                  />
                  <NumberField
                    aligned
                    control={control}
                    name="backoffFactor"
                    label="Backoff factor"
                    description="Multiplier applied to the open duration each trip."
                  />
                  <NumberField
                    aligned
                    control={control}
                    name="halfOpenProbes"
                    label="Half-open probes"
                    description="Trial requests allowed while half-open."
                  />
                  <NumberField
                    aligned
                    control={control}
                    name="halfOpenSuccesses"
                    label="Half-open successes"
                    description="Successful probes required to close the breaker."
                  />
                </div>
              </FieldGroup>
            </FieldSet>

            <FieldSeparator />

            <FieldSet>
              <FieldLegend className="flex flex-row gap-1 items-center">
                <SquareActivity size={16} />
                <p>Health checks &amp; TLS</p>
              </FieldLegend>
              <FieldGroup>
                <div className="grid gap-4 sm:grid-cols-2">
                  <TextField
                    control={control}
                    name="healthCheckPath"
                    label="Health check path"
                    placeholder="/health"
                    description="Path polled to determine backend health."
                  />
                  <NumberField
                    control={control}
                    name="healthInterval"
                    label="Health interval (seconds)"
                    description="How often to poll the health check path."
                  />
                </div>
                <SwitchField
                  control={control}
                  name="verifyTlsCert"
                  label="Verify TLS certificate"
                  description="Applies only when the base URL is https."
                />
              </FieldGroup>
            </FieldSet>

            <FieldSeparator />

            <FieldSet>
              <FieldLegend className="flex flex-row gap-1 items-center">
                <Tag size={16} />
                <p>Metadata</p>
              </FieldLegend>
              <FieldGroup>
                <Controller
                  control={control}
                  name="description"
                  render={({ field, fieldState }) => (
                    <Field data-invalid={fieldState.invalid}>
                      <FieldLabel htmlFor="description">Description</FieldLabel>
                      <Textarea
                        id="description"
                        name={field.name}
                        ref={field.ref}
                        placeholder="Optional notes about this backend"
                        aria-invalid={fieldState.invalid}
                        value={field.value ?? ''}
                        onBlur={field.onBlur}
                        onChange={(e) =>
                          field.onChange(
                            e.target.value === '' ? undefined : e.target.value,
                          )
                        }
                      />
                      <FieldError
                        errors={
                          fieldState.error ? [fieldState.error] : undefined
                        }
                      />
                    </Field>
                  )}
                />
                <TagField
                  control={control}
                  name="labels"
                  label="Labels"
                  placeholder="Type key=value and press Enter"
                  description="key=value labels for filtering and grouping."
                  validateItem={(value) =>
                    /^[^=\s]+=[^=]*$/.test(value)
                      ? null
                      : 'Use key=value format (no spaces).'
                  }
                />
              </FieldGroup>
            </FieldSet>
          </FieldGroup>
        </form>
      </CardContent>
    </Card>
  );
}

// ---------- Custom Field Components ----------
function TextField({
  control,
  name,
  label,
  description,
  placeholder,
  optional = false,
}: FieldProps & { placeholder?: string; optional?: boolean }) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <Field data-invalid={fieldState.invalid}>
          <FieldLabel htmlFor={name}>{label}</FieldLabel>
          <Input
            id={name}
            name={field.name}
            ref={field.ref}
            placeholder={placeholder}
            aria-invalid={fieldState.invalid}
            value={(field.value as string | undefined) ?? ''}
            onBlur={field.onBlur}
            onChange={(e) =>
              field.onChange(
                optional && e.target.value === '' ? undefined : e.target.value,
              )
            }
          />
          {description ? (
            <FieldDescription>{description}</FieldDescription>
          ) : null}
          <FieldError
            errors={fieldState.error ? [fieldState.error] : undefined}
          />
        </Field>
      )}
    />
  );
}

function NumberField({
  control,
  name,
  label,
  description,
  aligned = false,
}: FieldProps & { aligned?: boolean }) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => {
        const value = field.value;
        const display =
          typeof value === 'number' && !Number.isNaN(value) ? value : '';
        return (
          <Field
            data-invalid={fieldState.invalid}
            className={
              aligned
                ? 'sm:grid sm:grid-rows-subgrid sm:row-span-3 sm:gap-y-2'
                : undefined
            }
          >
            <FieldLabel htmlFor={name}>{label}</FieldLabel>
            <Input
              id={name}
              name={field.name}
              ref={field.ref}
              type="number"
              inputMode="numeric"
              aria-invalid={fieldState.invalid}
              value={display}
              onBlur={field.onBlur}
              onChange={(e) =>
                field.onChange(
                  e.target.value === '' ? undefined : e.target.valueAsNumber,
                )
              }
            />
            <div className="flex flex-col gap-1">
              {description ? (
                <FieldDescription>{description}</FieldDescription>
              ) : null}
              <FieldError
                errors={fieldState.error ? [fieldState.error] : undefined}
              />
            </div>
          </Field>
        );
      }}
    />
  );
}

function SwitchField({ control, name, label, description }: FieldProps) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <Field orientation="horizontal" data-invalid={fieldState.invalid}>
          <FieldContent>
            <FieldLabel htmlFor={name}>{label}</FieldLabel>
            {description ? (
              <FieldDescription>{description}</FieldDescription>
            ) : null}
          </FieldContent>
          <Switch
            id={name}
            checked={Boolean(field.value)}
            onCheckedChange={field.onChange}
          />
        </Field>
      )}
    />
  );
}

function TagField({
  control,
  name,
  label,
  description,
  placeholder,
  validateItem,
}: FieldProps & {
  placeholder?: string;
  validateItem?: (value: string) => string | null;
}) {
  const [draft, setDraft] = React.useState('');
  const [error, setError] = React.useState<string | null>(null);

  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => {
        const items = Array.isArray(field.value)
          ? (field.value as string[])
          : [];
        const invalid = fieldState.invalid || Boolean(error);

        const addItem = () => {
          const value = draft.trim();
          if (!value) return;
          const message = validateItem?.(value) ?? null;
          if (message) {
            setError(message);
            return;
          }
          if (items.includes(value)) {
            setError(`"${value}" is already added.`);
            return;
          }
          field.onChange([...items, value]);
          setDraft('');
          setError(null);
        };

        const removeItem = (item: string) => {
          field.onChange(items.filter((entry) => entry !== item));
        };

        return (
          <Field data-invalid={invalid}>
            <FieldLabel htmlFor={name}>{label}</FieldLabel>
            <InputGroup>
              <InputGroupInput
                id={name}
                name={field.name}
                ref={field.ref}
                placeholder={placeholder}
                aria-invalid={invalid}
                value={draft}
                onBlur={field.onBlur}
                onChange={(e) => {
                  setDraft(e.target.value);
                  if (error) setError(null);
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ',') {
                    e.preventDefault();
                    addItem();
                  } else if (
                    e.key === 'Backspace' &&
                    draft === '' &&
                    items.length
                  ) {
                    removeItem(items[items.length - 1]);
                  }
                }}
              />
              <InputGroupAddon align="inline-end">
                <InputGroupButton
                  size="icon-xs"
                  aria-label={`Add ${label.toLowerCase()}`}
                  disabled={draft.trim() === ''}
                  onClick={addItem}
                >
                  <CornerDownLeft />
                </InputGroupButton>
              </InputGroupAddon>
            </InputGroup>
            {items.length > 0 ? (
              <div className="flex flex-wrap gap-1.5 pt-1">
                {items.map((item) => (
                  <Badge key={item} variant="secondary" className="gap-1 pr-1">
                    {item}
                    <button
                      type="button"
                      aria-label={`Remove ${item}`}
                      onClick={() => removeItem(item)}
                      className="flex items-center rounded-full p-0.5 text-muted-foreground transition-colors hover:bg-foreground/10 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      <X size={12} />
                    </button>
                  </Badge>
                ))}
              </div>
            ) : null}
            {description ? (
              <FieldDescription>{description}</FieldDescription>
            ) : null}
            <FieldError
              errors={
                error
                  ? [{ message: error }]
                  : fieldState.error
                    ? [fieldState.error]
                    : undefined
              }
            />
          </Field>
        );
      }}
    />
  );
}
