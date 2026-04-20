// AI SDK schema factories. Three shapes can be handed to a tool's
// `inputSchema`: a raw Zod schema, a JSON Schema via `jsonSchema()`,
// or a FlexibleSchema. `asSchema()` normalizes any of those into the
// internal Schema type — useful when your code sees schemas from
// multiple sources and has to treat them uniformly.
import { jsonSchema, zodSchema, asSchema, z } from "ai";
import { output } from "kit";

const zs = zodSchema(z.object({ name: z.string(), age: z.number() }));
const js = jsonSchema({
  type: "object",
  properties: {
    name: { type: "string" },
    age: { type: "number" },
  },
  required: ["name", "age"],
});

const normFromZod = asSchema(zs);
const normFromJson = asSchema(js);

output({
  zodSchemaIsObject: typeof zs === "object" && zs !== null,
  jsonSchemaIsObject: typeof js === "object" && js !== null,
  asSchemaFromZodReturnsObject: typeof normFromZod === "object" && normFromZod !== null,
  asSchemaFromJsonReturnsObject: typeof normFromJson === "object" && normFromJson !== null,
  jsonSchemaHasJsonSchema: !!(js as any).jsonSchema,
  zodSchemaHasJsonSchema: !!(zs as any).jsonSchema,
});
