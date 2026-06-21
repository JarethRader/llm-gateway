import { z } from "zod";
import { ModelSchema } from "./ModelFormSchema";

export const Registry = z.registry<{ description: string }>();

Registry.add(ModelSchema, { description: "Model Form schema" });
