package kit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
)

// AIDomain handles AI model operations: generate, embed, embedMany, generateObject.
type AIDomain struct {
	kit *Kernel
}

func newAIDomain(k *Kernel) *AIDomain {
	return &AIDomain{kit: k}
}

func (d *AIDomain) Generate(ctx context.Context, req messages.AiGenerateMsg) (*messages.AiGenerateResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__ai_generate.ts", `
		var req = globalThis.__pending_req;
		var result = await ai.generate(req);
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("ai.generate: %w", err)
	}
	var resp messages.AiGenerateResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("ai.generate: unmarshal: %w", err)
	}
	return &resp, nil
}

func (d *AIDomain) Embed(ctx context.Context, req messages.AiEmbedMsg) (*messages.AiEmbedResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__ai_embed.ts", `
		var req = globalThis.__pending_req;
		var result = await ai.embed(req);
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("ai.embed: %w", err)
	}
	var resp messages.AiEmbedResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("ai.embed: unmarshal: %w", err)
	}
	return &resp, nil
}

func (d *AIDomain) EmbedMany(ctx context.Context, req messages.AiEmbedManyMsg) (*messages.AiEmbedManyResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__ai_embedmany.ts", `
		var req = globalThis.__pending_req;
		var result = await ai.embedMany(req);
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("ai.embedMany: %w", err)
	}
	var resp messages.AiEmbedManyResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("ai.embedMany: unmarshal: %w", err)
	}
	return &resp, nil
}

func (d *AIDomain) GenerateObject(ctx context.Context, req messages.AiGenerateObjectMsg) (*messages.AiGenerateObjectResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__ai_generateobject.ts", `
		var req = globalThis.__pending_req;
		if (req.schema && typeof req.schema === "object" && !req.schema._def) {
			function jsonToZod(s) {
				if (!s || typeof s !== "object") return z.any();
				if (s.type === "string") return z.string();
				if (s.type === "number" || s.type === "integer") return z.number();
				if (s.type === "boolean") return z.boolean();
				if (s.type === "array") return z.array(jsonToZod(s.items));
				if (s.type === "object" && s.properties) {
					var shape = {};
					var required = s.required || [];
					for (var key in s.properties) {
						var field = jsonToZod(s.properties[key]);
						if (s.properties[key].description) field = field.describe(s.properties[key].description);
						if (required.indexOf(key) < 0) field = field.optional();
						shape[key] = field;
					}
					return z.object(shape);
				}
				return z.any();
			}
			req.schema = jsonToZod(req.schema);
		}
		var result = await ai.generateObject(req);
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("ai.generateObject: %w", err)
	}
	var resp messages.AiGenerateObjectResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("ai.generateObject: unmarshal: %w", err)
	}
	return &resp, nil
}
