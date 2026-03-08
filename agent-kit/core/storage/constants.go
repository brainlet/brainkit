// Ported from: packages/core/src/storage/constants.ts
package storage

// ---------------------------------------------------------------------------
// Table name constants
// ---------------------------------------------------------------------------

// TableName is a string type for table name constants.
type TableName = string

const (
	TableWorkflowSnapshot         TableName = "mastra_workflow_snapshot"
	TableMessages                 TableName = "mastra_messages"
	TableThreads                  TableName = "mastra_threads"
	TableTraces                   TableName = "mastra_traces"
	TableResources                TableName = "mastra_resources"
	TableScorers                  TableName = "mastra_scorers"
	TableSpans                    TableName = "mastra_ai_spans"
	TableAgents                   TableName = "mastra_agents"
	TableAgentVersions            TableName = "mastra_agent_versions"
	TableObservationalMemory      TableName = "mastra_observational_memory"
	TablePromptBlocks             TableName = "mastra_prompt_blocks"
	TablePromptBlockVersions      TableName = "mastra_prompt_block_versions"
	TableScorerDefinitions        TableName = "mastra_scorer_definitions"
	TableScorerDefinitionVersions TableName = "mastra_scorer_definition_versions"
	TableMCPClients               TableName = "mastra_mcp_clients"
	TableMCPClientVersions        TableName = "mastra_mcp_client_versions"
	TableMCPServers               TableName = "mastra_mcp_servers"
	TableMCPServerVersions        TableName = "mastra_mcp_server_versions"
	TableWorkspaces               TableName = "mastra_workspaces"
	TableWorkspaceVersions        TableName = "mastra_workspace_versions"
	TableSkills                   TableName = "mastra_skills"
	TableSkillVersions            TableName = "mastra_skill_versions"
	TableSkillBlobs               TableName = "mastra_skill_blobs"
	// Dataset tables
	TableDatasets        TableName = "mastra_datasets"
	TableDatasetItems    TableName = "mastra_dataset_items"
	TableDatasetVersions TableName = "mastra_dataset_versions"
	// Experiment tables
	TableExperiments       TableName = "mastra_experiments"
	TableExperimentResults TableName = "mastra_experiment_results"
)

// ---------------------------------------------------------------------------
// StorageColumn describes a single column in a storage table schema.
// ---------------------------------------------------------------------------

// StorageColumnType enumerates the column types recognised by the schema.
type StorageColumnType = string

const (
	ColTypeText      StorageColumnType = "text"
	ColTypeTimestamp  StorageColumnType = "timestamp"
	ColTypeUUID      StorageColumnType = "uuid"
	ColTypeJSONB     StorageColumnType = "jsonb"
	ColTypeInteger   StorageColumnType = "integer"
	ColTypeFloat     StorageColumnType = "float"
	ColTypeBigint    StorageColumnType = "bigint"
	ColTypeBoolean   StorageColumnType = "boolean"
)

// StorageColumnRef describes a foreign-key reference.
type StorageColumnRef struct {
	Table  string
	Column string
}

// StorageColumn describes a single column's type and constraints.
type StorageColumn struct {
	Type       StorageColumnType
	PrimaryKey bool
	Nullable   bool
	References *StorageColumnRef // nil when there is no FK reference
}

// StorageTableConfig provides table-level configuration such as composite
// primary keys. Tables not listed in TableConfigs use single-column PKs from
// their schema.
type StorageTableConfig struct {
	Columns             map[string]StorageColumn
	CompositePrimaryKey []string
}

// ---------------------------------------------------------------------------
// Schema definitions – one map per table
// ---------------------------------------------------------------------------

// ScorersSchema defines the column schema for the scorers table.
var ScorersSchema = map[string]StorageColumn{
	"id":                   {Type: ColTypeText, PrimaryKey: true},
	"scorerId":             {Type: ColTypeText},
	"traceId":              {Type: ColTypeText, Nullable: true},
	"spanId":               {Type: ColTypeText, Nullable: true},
	"runId":                {Type: ColTypeText},
	"scorer":               {Type: ColTypeJSONB},
	"preprocessStepResult": {Type: ColTypeJSONB, Nullable: true},
	"extractStepResult":    {Type: ColTypeJSONB, Nullable: true},
	"analyzeStepResult":    {Type: ColTypeJSONB, Nullable: true},
	"score":                {Type: ColTypeFloat},
	"reason":               {Type: ColTypeText, Nullable: true},
	"metadata":             {Type: ColTypeJSONB, Nullable: true},
	"preprocessPrompt":     {Type: ColTypeText, Nullable: true},
	"extractPrompt":        {Type: ColTypeText, Nullable: true},
	"generateScorePrompt":  {Type: ColTypeText, Nullable: true},
	"generateReasonPrompt": {Type: ColTypeText, Nullable: true},
	"analyzePrompt":        {Type: ColTypeText, Nullable: true},
	// Deprecated
	"reasonPrompt":      {Type: ColTypeText, Nullable: true},
	"input":             {Type: ColTypeJSONB},
	"output":            {Type: ColTypeJSONB},              // MESSAGE OUTPUT
	"additionalContext":  {Type: ColTypeJSONB, Nullable: true}, // DATA FROM THE CONTEXT PARAM ON AN AGENT
	"requestContext":     {Type: ColTypeJSONB, Nullable: true}, // THE EVALUATE Request Context FOR THE RUN
	"entityType":        {Type: ColTypeText, Nullable: true},   // WORKFLOW, AGENT, TOOL, STEP, NETWORK
	"entity":            {Type: ColTypeJSONB, Nullable: true},  // MINIMAL JSON DATA ABOUT ENTITY
	"entityId":          {Type: ColTypeText, Nullable: true},
	"source":            {Type: ColTypeText},
	"resourceId":        {Type: ColTypeText, Nullable: true},
	"threadId":          {Type: ColTypeText, Nullable: true},
	"createdAt":         {Type: ColTypeTimestamp},
	"updatedAt":         {Type: ColTypeTimestamp},
}

// SpanSchema is the result of buildStorageSchema(spanRecordSchema).
//
// The TypeScript source derives this dynamically from the Zod spanRecordSchema.
// Here we materialise the result statically so that no Zod runtime is needed.
//
// TODO: If the upstream spanRecordSchema changes, this must be updated manually.
var SpanSchema = map[string]StorageColumn{
	// Required identifiers
	"traceId":  {Type: ColTypeText},
	"spanId":   {Type: ColTypeText},
	"name":     {Type: ColTypeText},
	"spanType": {Type: ColTypeText}, // nativeEnum → text
	"isEvent":  {Type: ColTypeBoolean},
	"startedAt": {Type: ColTypeTimestamp},
	// Shared optional fields – entity identification
	"parentSpanId": {Type: ColTypeText, Nullable: true},
	"entityType":   {Type: ColTypeText, Nullable: true},
	"entityId":     {Type: ColTypeText, Nullable: true},
	"entityName":   {Type: ColTypeText, Nullable: true},
	// Identity & tenancy
	"userId":         {Type: ColTypeText, Nullable: true},
	"organizationId": {Type: ColTypeText, Nullable: true},
	"resourceId":     {Type: ColTypeText, Nullable: true},
	// Correlation IDs
	"runId":     {Type: ColTypeText, Nullable: true},
	"sessionId": {Type: ColTypeText, Nullable: true},
	"threadId":  {Type: ColTypeText, Nullable: true},
	"requestId": {Type: ColTypeText, Nullable: true},
	// Deployment context
	"environment": {Type: ColTypeText, Nullable: true},
	"source":      {Type: ColTypeText, Nullable: true},
	"serviceName": {Type: ColTypeText, Nullable: true},
	"scope":       {Type: ColTypeJSONB, Nullable: true},
	// Filterable data
	"metadata": {Type: ColTypeJSONB, Nullable: true},
	"tags":     {Type: ColTypeJSONB, Nullable: true},
	// Additional span-specific fields
	"attributes": {Type: ColTypeJSONB, Nullable: true},
	"links":      {Type: ColTypeJSONB, Nullable: true},
	"input":      {Type: ColTypeJSONB, Nullable: true},
	"output":     {Type: ColTypeJSONB, Nullable: true},
	"error":      {Type: ColTypeJSONB, Nullable: true},
	"endedAt":    {Type: ColTypeTimestamp, Nullable: true},
	// Database timestamps
	"createdAt": {Type: ColTypeTimestamp},
	"updatedAt": {Type: ColTypeTimestamp, Nullable: true},
}

// OldSpanSchema is the legacy span schema retained for migration purposes.
//
// Deprecated: Use SpanSchema instead.
var OldSpanSchema = map[string]StorageColumn{
	"traceId":      {Type: ColTypeText},
	"spanId":       {Type: ColTypeText},
	"parentSpanId": {Type: ColTypeText, Nullable: true},
	"name":         {Type: ColTypeText},
	"scope":        {Type: ColTypeJSONB, Nullable: true},
	"spanType":     {Type: ColTypeText},
	"attributes":   {Type: ColTypeJSONB, Nullable: true},
	"metadata":     {Type: ColTypeJSONB, Nullable: true},
	"links":        {Type: ColTypeJSONB, Nullable: true},
	"input":        {Type: ColTypeJSONB, Nullable: true},
	"output":       {Type: ColTypeJSONB, Nullable: true},
	"error":        {Type: ColTypeJSONB, Nullable: true},
	"startedAt":    {Type: ColTypeTimestamp},
	"endedAt":      {Type: ColTypeTimestamp, Nullable: true},
	"createdAt":    {Type: ColTypeTimestamp},
	"updatedAt":    {Type: ColTypeTimestamp, Nullable: true},
	"isEvent":      {Type: ColTypeBoolean},
}

// AgentsSchema defines the column schema for the agents table.
var AgentsSchema = map[string]StorageColumn{
	"id":              {Type: ColTypeText, PrimaryKey: true},
	"status":          {Type: ColTypeText},          // 'draft' or 'published'
	"activeVersionId": {Type: ColTypeText, Nullable: true}, // FK to agent_versions.id
	"authorId":        {Type: ColTypeText, Nullable: true},
	"metadata":        {Type: ColTypeJSONB, Nullable: true},
	"createdAt":       {Type: ColTypeTimestamp},
	"updatedAt":       {Type: ColTypeTimestamp},
}

// AgentVersionsSchema defines the column schema for the agent_versions table.
var AgentVersionsSchema = map[string]StorageColumn{
	"id":                   {Type: ColTypeText, PrimaryKey: true},
	"agentId":              {Type: ColTypeText},
	"versionNumber":        {Type: ColTypeInteger},
	"name":                 {Type: ColTypeText},
	"description":          {Type: ColTypeText, Nullable: true},
	"instructions":         {Type: ColTypeText},
	"model":                {Type: ColTypeJSONB},
	"tools":                {Type: ColTypeJSONB, Nullable: true},
	"defaultOptions":       {Type: ColTypeJSONB, Nullable: true},
	"workflows":            {Type: ColTypeJSONB, Nullable: true},
	"agents":               {Type: ColTypeJSONB, Nullable: true},
	"integrationTools":     {Type: ColTypeJSONB, Nullable: true},
	"inputProcessors":      {Type: ColTypeJSONB, Nullable: true},
	"outputProcessors":     {Type: ColTypeJSONB, Nullable: true},
	"memory":               {Type: ColTypeJSONB, Nullable: true},
	"scorers":              {Type: ColTypeJSONB, Nullable: true},
	"mcpClients":           {Type: ColTypeJSONB, Nullable: true},
	"requestContextSchema": {Type: ColTypeJSONB, Nullable: true},
	"workspace":            {Type: ColTypeJSONB, Nullable: true},
	"skills":               {Type: ColTypeJSONB, Nullable: true},
	"skillsFormat":         {Type: ColTypeText, Nullable: true},
	"changedFields":        {Type: ColTypeJSONB, Nullable: true},
	"changeMessage":        {Type: ColTypeText, Nullable: true},
	"createdAt":            {Type: ColTypeTimestamp},
}

// PromptBlocksSchema defines the column schema for the prompt_blocks table.
var PromptBlocksSchema = map[string]StorageColumn{
	"id":              {Type: ColTypeText, PrimaryKey: true},
	"status":          {Type: ColTypeText},          // 'draft', 'published', or 'archived'
	"activeVersionId": {Type: ColTypeText, Nullable: true},
	"authorId":        {Type: ColTypeText, Nullable: true},
	"metadata":        {Type: ColTypeJSONB, Nullable: true},
	"createdAt":       {Type: ColTypeTimestamp},
	"updatedAt":       {Type: ColTypeTimestamp},
}

// PromptBlockVersionsSchema defines the column schema for the prompt_block_versions table.
var PromptBlockVersionsSchema = map[string]StorageColumn{
	"id":                   {Type: ColTypeText, PrimaryKey: true},
	"blockId":              {Type: ColTypeText},
	"versionNumber":        {Type: ColTypeInteger},
	"name":                 {Type: ColTypeText},
	"description":          {Type: ColTypeText, Nullable: true},
	"content":              {Type: ColTypeText},
	"rules":                {Type: ColTypeJSONB, Nullable: true},
	"requestContextSchema": {Type: ColTypeJSONB, Nullable: true},
	"changedFields":        {Type: ColTypeJSONB, Nullable: true},
	"changeMessage":        {Type: ColTypeText, Nullable: true},
	"createdAt":            {Type: ColTypeTimestamp},
}

// ScorerDefinitionsSchema defines the column schema for the scorer_definitions table.
var ScorerDefinitionsSchema = map[string]StorageColumn{
	"id":              {Type: ColTypeText, PrimaryKey: true},
	"status":          {Type: ColTypeText},
	"activeVersionId": {Type: ColTypeText, Nullable: true},
	"authorId":        {Type: ColTypeText, Nullable: true},
	"metadata":        {Type: ColTypeJSONB, Nullable: true},
	"createdAt":       {Type: ColTypeTimestamp},
	"updatedAt":       {Type: ColTypeTimestamp},
}

// ScorerDefinitionVersionsSchema defines the column schema for the scorer_definition_versions table.
var ScorerDefinitionVersionsSchema = map[string]StorageColumn{
	"id":                  {Type: ColTypeText, PrimaryKey: true},
	"scorerDefinitionId":  {Type: ColTypeText},
	"versionNumber":       {Type: ColTypeInteger},
	"name":                {Type: ColTypeText},
	"description":         {Type: ColTypeText, Nullable: true},
	"type":                {Type: ColTypeText},
	"model":               {Type: ColTypeJSONB, Nullable: true},
	"instructions":        {Type: ColTypeText, Nullable: true},
	"scoreRange":          {Type: ColTypeJSONB, Nullable: true},
	"presetConfig":        {Type: ColTypeJSONB, Nullable: true},
	"defaultSampling":     {Type: ColTypeJSONB, Nullable: true},
	"changedFields":       {Type: ColTypeJSONB, Nullable: true},
	"changeMessage":       {Type: ColTypeText, Nullable: true},
	"createdAt":           {Type: ColTypeTimestamp},
}

// MCPClientsSchema defines the column schema for the mcp_clients table.
var MCPClientsSchema = map[string]StorageColumn{
	"id":              {Type: ColTypeText, PrimaryKey: true},
	"status":          {Type: ColTypeText},
	"activeVersionId": {Type: ColTypeText, Nullable: true},
	"authorId":        {Type: ColTypeText, Nullable: true},
	"metadata":        {Type: ColTypeJSONB, Nullable: true},
	"createdAt":       {Type: ColTypeTimestamp},
	"updatedAt":       {Type: ColTypeTimestamp},
}

// MCPClientVersionsSchema defines the column schema for the mcp_client_versions table.
var MCPClientVersionsSchema = map[string]StorageColumn{
	"id":            {Type: ColTypeText, PrimaryKey: true},
	"mcpClientId":   {Type: ColTypeText},
	"versionNumber": {Type: ColTypeInteger},
	"name":          {Type: ColTypeText},
	"description":   {Type: ColTypeText, Nullable: true},
	"servers":       {Type: ColTypeJSONB},
	"changedFields": {Type: ColTypeJSONB, Nullable: true},
	"changeMessage": {Type: ColTypeText, Nullable: true},
	"createdAt":     {Type: ColTypeTimestamp},
}

// MCPServersSchema defines the column schema for the mcp_servers table.
var MCPServersSchema = map[string]StorageColumn{
	"id":              {Type: ColTypeText, PrimaryKey: true},
	"status":          {Type: ColTypeText},
	"activeVersionId": {Type: ColTypeText, Nullable: true},
	"authorId":        {Type: ColTypeText, Nullable: true},
	"metadata":        {Type: ColTypeJSONB, Nullable: true},
	"createdAt":       {Type: ColTypeTimestamp},
	"updatedAt":       {Type: ColTypeTimestamp},
}

// MCPServerVersionsSchema defines the column schema for the mcp_server_versions table.
var MCPServerVersionsSchema = map[string]StorageColumn{
	"id":               {Type: ColTypeText, PrimaryKey: true},
	"mcpServerId":      {Type: ColTypeText},
	"versionNumber":    {Type: ColTypeInteger},
	"name":             {Type: ColTypeText},
	"version":          {Type: ColTypeText},
	"description":      {Type: ColTypeText, Nullable: true},
	"instructions":     {Type: ColTypeText, Nullable: true},
	"repository":       {Type: ColTypeJSONB, Nullable: true},
	"releaseDate":      {Type: ColTypeText, Nullable: true},
	"isLatest":         {Type: ColTypeBoolean, Nullable: true},
	"packageCanonical": {Type: ColTypeText, Nullable: true},
	"tools":            {Type: ColTypeJSONB, Nullable: true},
	"agents":           {Type: ColTypeJSONB, Nullable: true},
	"workflows":        {Type: ColTypeJSONB, Nullable: true},
	"changedFields":    {Type: ColTypeJSONB, Nullable: true},
	"changeMessage":    {Type: ColTypeText, Nullable: true},
	"createdAt":        {Type: ColTypeTimestamp},
}

// WorkspacesSchema defines the column schema for the workspaces table.
var WorkspacesSchema = map[string]StorageColumn{
	"id":              {Type: ColTypeText, PrimaryKey: true},
	"status":          {Type: ColTypeText},
	"activeVersionId": {Type: ColTypeText, Nullable: true},
	"authorId":        {Type: ColTypeText, Nullable: true},
	"metadata":        {Type: ColTypeJSONB, Nullable: true},
	"createdAt":       {Type: ColTypeTimestamp},
	"updatedAt":       {Type: ColTypeTimestamp},
}

// WorkspaceVersionsSchema defines the column schema for the workspace_versions table.
var WorkspaceVersionsSchema = map[string]StorageColumn{
	"id":               {Type: ColTypeText, PrimaryKey: true},
	"workspaceId":      {Type: ColTypeText},
	"versionNumber":    {Type: ColTypeInteger},
	"name":             {Type: ColTypeText},
	"description":      {Type: ColTypeText, Nullable: true},
	"filesystem":       {Type: ColTypeJSONB, Nullable: true},
	"sandbox":          {Type: ColTypeJSONB, Nullable: true},
	"mounts":           {Type: ColTypeJSONB, Nullable: true},
	"search":           {Type: ColTypeJSONB, Nullable: true},
	"skills":           {Type: ColTypeJSONB, Nullable: true},
	"tools":            {Type: ColTypeJSONB, Nullable: true},
	"autoSync":         {Type: ColTypeBoolean, Nullable: true},
	"operationTimeout": {Type: ColTypeInteger, Nullable: true},
	"changedFields":    {Type: ColTypeJSONB, Nullable: true},
	"changeMessage":    {Type: ColTypeText, Nullable: true},
	"createdAt":        {Type: ColTypeTimestamp},
}

// SkillsSchema defines the column schema for the skills table.
var SkillsSchema = map[string]StorageColumn{
	"id":              {Type: ColTypeText, PrimaryKey: true},
	"status":          {Type: ColTypeText},
	"activeVersionId": {Type: ColTypeText, Nullable: true},
	"authorId":        {Type: ColTypeText, Nullable: true},
	"createdAt":       {Type: ColTypeTimestamp},
	"updatedAt":       {Type: ColTypeTimestamp},
}

// SkillVersionsSchema defines the column schema for the skill_versions table.
var SkillVersionsSchema = map[string]StorageColumn{
	"id":            {Type: ColTypeText, PrimaryKey: true},
	"skillId":       {Type: ColTypeText},
	"versionNumber": {Type: ColTypeInteger},
	"name":          {Type: ColTypeText},
	"description":   {Type: ColTypeText},
	"instructions":  {Type: ColTypeText},
	"license":       {Type: ColTypeText, Nullable: true},
	"compatibility": {Type: ColTypeJSONB, Nullable: true},
	"source":        {Type: ColTypeJSONB, Nullable: true},
	"references":    {Type: ColTypeJSONB, Nullable: true},
	"scripts":       {Type: ColTypeJSONB, Nullable: true},
	"assets":        {Type: ColTypeJSONB, Nullable: true},
	"metadata":      {Type: ColTypeJSONB, Nullable: true},
	"tree":          {Type: ColTypeJSONB, Nullable: true},
	"changedFields": {Type: ColTypeJSONB, Nullable: true},
	"changeMessage": {Type: ColTypeText, Nullable: true},
	"createdAt":     {Type: ColTypeTimestamp},
}

// SkillBlobsSchema defines the column schema for the skill_blobs table.
var SkillBlobsSchema = map[string]StorageColumn{
	"hash":      {Type: ColTypeText, PrimaryKey: true},
	"content":   {Type: ColTypeText},
	"size":      {Type: ColTypeInteger},
	"mimeType":  {Type: ColTypeText, Nullable: true},
	"createdAt": {Type: ColTypeTimestamp},
}

// ObservationalMemorySchema defines the column schema for the observational_memory table.
var ObservationalMemorySchema = map[string]StorageColumn{
	"id":                                {Type: ColTypeText, PrimaryKey: true},
	"lookupKey":                         {Type: ColTypeText},
	"scope":                             {Type: ColTypeText},
	"resourceId":                        {Type: ColTypeText, Nullable: true},
	"threadId":                          {Type: ColTypeText, Nullable: true},
	"activeObservations":                {Type: ColTypeText},
	"activeObservationsPendingUpdate":    {Type: ColTypeText, Nullable: true},
	"originType":                        {Type: ColTypeText},
	"config":                            {Type: ColTypeText},
	"generationCount":                   {Type: ColTypeInteger},
	"lastObservedAt":                    {Type: ColTypeTimestamp, Nullable: true},
	"lastReflectionAt":                  {Type: ColTypeTimestamp, Nullable: true},
	"pendingMessageTokens":              {Type: ColTypeInteger},
	"totalTokensObserved":               {Type: ColTypeInteger},
	"observationTokenCount":             {Type: ColTypeInteger},
	"isObserving":                       {Type: ColTypeBoolean},
	"isReflecting":                      {Type: ColTypeBoolean},
	"observedMessageIds":                {Type: ColTypeJSONB, Nullable: true},
	"observedTimezone":                  {Type: ColTypeText, Nullable: true},
	"bufferedObservations":              {Type: ColTypeText, Nullable: true},
	"bufferedObservationTokens":         {Type: ColTypeInteger, Nullable: true},
	"bufferedMessageIds":                {Type: ColTypeJSONB, Nullable: true},
	"bufferedReflection":                {Type: ColTypeText, Nullable: true},
	"bufferedReflectionTokens":          {Type: ColTypeInteger, Nullable: true},
	"bufferedReflectionInputTokens":     {Type: ColTypeInteger, Nullable: true},
	"reflectedObservationLineCount":     {Type: ColTypeInteger, Nullable: true},
	"bufferedObservationChunks":         {Type: ColTypeJSONB, Nullable: true},
	"isBufferingObservation":            {Type: ColTypeBoolean},
	"isBufferingReflection":             {Type: ColTypeBoolean},
	"lastBufferedAtTokens":              {Type: ColTypeInteger},
	"lastBufferedAtTime":                {Type: ColTypeTimestamp, Nullable: true},
	"metadata":                          {Type: ColTypeJSONB, Nullable: true},
	"createdAt":                         {Type: ColTypeTimestamp},
	"updatedAt":                         {Type: ColTypeTimestamp},
}

// DatasetsSchema defines the column schema for the datasets table.
var DatasetsSchema = map[string]StorageColumn{
	"id":               {Type: ColTypeText, PrimaryKey: true},
	"name":             {Type: ColTypeText},
	"description":      {Type: ColTypeText, Nullable: true},
	"metadata":         {Type: ColTypeJSONB, Nullable: true},
	"inputSchema":      {Type: ColTypeJSONB, Nullable: true},
	"groundTruthSchema": {Type: ColTypeJSONB, Nullable: true},
	"version":          {Type: ColTypeInteger},
	"createdAt":        {Type: ColTypeTimestamp},
	"updatedAt":        {Type: ColTypeTimestamp},
}

// DatasetItemsSchema defines the column schema for the dataset_items table.
var DatasetItemsSchema = map[string]StorageColumn{
	"id":             {Type: ColTypeText},
	"datasetId":      {Type: ColTypeText, References: &StorageColumnRef{Table: "mastra_datasets", Column: "id"}},
	"datasetVersion": {Type: ColTypeInteger},
	"validTo":        {Type: ColTypeInteger, Nullable: true},
	"isDeleted":      {Type: ColTypeBoolean},
	"input":          {Type: ColTypeJSONB},
	"groundTruth":    {Type: ColTypeJSONB, Nullable: true},
	"metadata":       {Type: ColTypeJSONB, Nullable: true},
	"createdAt":      {Type: ColTypeTimestamp},
	"updatedAt":      {Type: ColTypeTimestamp},
}

// DatasetVersionsSchema defines the column schema for the dataset_versions table.
var DatasetVersionsSchema = map[string]StorageColumn{
	"id":        {Type: ColTypeText, PrimaryKey: true},
	"datasetId": {Type: ColTypeText, References: &StorageColumnRef{Table: "mastra_datasets", Column: "id"}},
	"version":   {Type: ColTypeInteger},
	"createdAt": {Type: ColTypeTimestamp},
}

// ExperimentsSchema defines the column schema for the experiments table.
var ExperimentsSchema = map[string]StorageColumn{
	"id":             {Type: ColTypeText, PrimaryKey: true},
	"name":           {Type: ColTypeText, Nullable: true},
	"description":    {Type: ColTypeText, Nullable: true},
	"metadata":       {Type: ColTypeJSONB, Nullable: true},
	"datasetId":      {Type: ColTypeText, Nullable: true, References: &StorageColumnRef{Table: "mastra_datasets", Column: "id"}},
	"datasetVersion": {Type: ColTypeInteger, Nullable: true},
	"targetType":     {Type: ColTypeText},
	"targetId":       {Type: ColTypeText},
	"status":         {Type: ColTypeText},
	"totalItems":     {Type: ColTypeInteger},
	"succeededCount": {Type: ColTypeInteger},
	"failedCount":    {Type: ColTypeInteger},
	"skippedCount":   {Type: ColTypeInteger},
	"startedAt":      {Type: ColTypeTimestamp, Nullable: true},
	"completedAt":    {Type: ColTypeTimestamp, Nullable: true},
	"createdAt":      {Type: ColTypeTimestamp},
	"updatedAt":      {Type: ColTypeTimestamp},
}

// ExperimentResultsSchema defines the column schema for the experiment_results table.
var ExperimentResultsSchema = map[string]StorageColumn{
	"id":                 {Type: ColTypeText, PrimaryKey: true},
	"experimentId":       {Type: ColTypeText, References: &StorageColumnRef{Table: "mastra_experiments", Column: "id"}},
	"itemId":             {Type: ColTypeText, References: &StorageColumnRef{Table: "mastra_dataset_items", Column: "id"}},
	"itemDatasetVersion": {Type: ColTypeInteger, Nullable: true},
	"input":              {Type: ColTypeJSONB},
	"output":             {Type: ColTypeJSONB, Nullable: true},
	"groundTruth":        {Type: ColTypeJSONB, Nullable: true},
	"error":              {Type: ColTypeJSONB, Nullable: true},
	"startedAt":          {Type: ColTypeTimestamp},
	"completedAt":        {Type: ColTypeTimestamp},
	"retryCount":         {Type: ColTypeInteger},
	"traceId":            {Type: ColTypeText, Nullable: true},
	"createdAt":          {Type: ColTypeTimestamp},
}

// ---------------------------------------------------------------------------
// TABLE_SCHEMAS – master mapping from table name → column schema
// ---------------------------------------------------------------------------

// TableSchemas maps every TableName to its column schema.
var TableSchemas = map[TableName]map[string]StorageColumn{
	TableWorkflowSnapshot: {
		"workflow_name": {Type: ColTypeText},
		"run_id":        {Type: ColTypeText},
		"resourceId":    {Type: ColTypeText, Nullable: true},
		"snapshot":      {Type: ColTypeJSONB},
		"createdAt":     {Type: ColTypeTimestamp},
		"updatedAt":     {Type: ColTypeTimestamp},
	},
	TableScorers: ScorersSchema,
	TableThreads: {
		"id":         {Type: ColTypeText, PrimaryKey: true},
		"resourceId": {Type: ColTypeText},
		"title":      {Type: ColTypeText},
		"metadata":   {Type: ColTypeJSONB, Nullable: true},
		"createdAt":  {Type: ColTypeTimestamp},
		"updatedAt":  {Type: ColTypeTimestamp},
	},
	TableMessages: {
		"id":         {Type: ColTypeText, PrimaryKey: true},
		"thread_id":  {Type: ColTypeText},
		"content":    {Type: ColTypeText},
		"role":       {Type: ColTypeText},
		"type":       {Type: ColTypeText},
		"createdAt":  {Type: ColTypeTimestamp},
		"resourceId": {Type: ColTypeText, Nullable: true},
	},
	TableSpans:  SpanSchema,
	TableTraces: {
		"id":           {Type: ColTypeText, PrimaryKey: true},
		"parentSpanId": {Type: ColTypeText, Nullable: true},
		"name":         {Type: ColTypeText},
		"traceId":      {Type: ColTypeText},
		"scope":        {Type: ColTypeText},
		"kind":         {Type: ColTypeInteger},
		"attributes":   {Type: ColTypeJSONB, Nullable: true},
		"status":       {Type: ColTypeJSONB, Nullable: true},
		"events":       {Type: ColTypeJSONB, Nullable: true},
		"links":        {Type: ColTypeJSONB, Nullable: true},
		"other":        {Type: ColTypeText, Nullable: true},
		"startTime":    {Type: ColTypeBigint},
		"endTime":      {Type: ColTypeBigint},
		"createdAt":    {Type: ColTypeTimestamp},
	},
	TableResources: {
		"id":            {Type: ColTypeText, PrimaryKey: true},
		"workingMemory": {Type: ColTypeText, Nullable: true},
		"metadata":      {Type: ColTypeJSONB, Nullable: true},
		"createdAt":     {Type: ColTypeTimestamp},
		"updatedAt":     {Type: ColTypeTimestamp},
	},
	TableAgents:                   AgentsSchema,
	TableAgentVersions:            AgentVersionsSchema,
	TablePromptBlocks:             PromptBlocksSchema,
	TablePromptBlockVersions:      PromptBlockVersionsSchema,
	TableScorerDefinitions:        ScorerDefinitionsSchema,
	TableScorerDefinitionVersions: ScorerDefinitionVersionsSchema,
	TableMCPClients:               MCPClientsSchema,
	TableMCPClientVersions:        MCPClientVersionsSchema,
	TableMCPServers:               MCPServersSchema,
	TableMCPServerVersions:        MCPServerVersionsSchema,
	TableWorkspaces:               WorkspacesSchema,
	TableWorkspaceVersions:        WorkspaceVersionsSchema,
	TableSkills:                   SkillsSchema,
	TableSkillVersions:            SkillVersionsSchema,
	TableSkillBlobs:               SkillBlobsSchema,
	TableDatasets:                 DatasetsSchema,
	TableDatasetItems:             DatasetItemsSchema,
	TableDatasetVersions:          DatasetVersionsSchema,
	TableExperiments:              ExperimentsSchema,
	TableExperimentResults:        ExperimentResultsSchema,
}

// TableConfigs provides table-level config for tables that need composite
// primary keys or other table-level settings. Tables not listed here use
// single-column PKs from their schema.
var TableConfigs = map[TableName]StorageTableConfig{
	TableDatasetItems: {
		Columns:             DatasetItemsSchema,
		CompositePrimaryKey: []string{"id", "datasetVersion"},
	},
}

// ObservationalMemoryTableSchema is exported separately because observational
// memory is optional and not part of the core TableSchemas.
var ObservationalMemoryTableSchema = map[TableName]map[string]StorageColumn{
	TableObservationalMemory: ObservationalMemorySchema,
}
