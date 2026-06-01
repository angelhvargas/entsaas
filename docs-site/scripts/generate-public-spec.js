import fs from 'fs'
import path from 'path'
import yaml from 'js-yaml'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const rootDir = path.resolve(__dirname, '../../')

// Support overriding the input path via env var (used in Docker builds)
const inputPath = process.env.OPENAPI_SOURCE_PATH || path.join(rootDir, 'docs/api/openapi.yaml')

// Ensure directory exists for output
const publicDir = path.join(__dirname, '../src/public')
if (!fs.existsSync(publicDir)) {
  fs.mkdirSync(publicDir, { recursive: true })
}
const outputPath = path.join(publicDir, 'public-api.yaml')

console.log(`Reading spec from ${inputPath}`)
const fileContents = fs.readFileSync(inputPath, 'utf8')
const doc = yaml.load(fileContents)

// Filter paths to only include public-preview (or other public) routes
const publicPaths = {}
let routeCount = 0

for (const [pathStr, pathObj] of Object.entries(doc.paths)) {
  const publicMethods = {}
  let hasPublicMethods = false

  for (const [method, operation] of Object.entries(pathObj)) {
    if (method === 'parameters') continue // Skip path-level params

    // Check the x-exposure extension
    const exposure = operation['x-exposure']
    if (exposure === 'public-preview' || exposure === 'public-ga') {
      publicMethods[method] = operation
      hasPublicMethods = true
      routeCount++
    }
  }

  if (hasPublicMethods) {
    publicPaths[pathStr] = publicMethods
    // Copy path-level params if they exist
    if (pathObj.parameters) {
      publicPaths[pathStr].parameters = pathObj.parameters
    }
  }
}

doc.paths = publicPaths
doc.info.title = 'EntSaaS Public API'
doc.info.description = 'Public REST API reference for the EntSaaS Multi-Tenant Framework.'

// Clean up tags that are no longer referenced
const usedTags = new Set()
for (const pathObj of Object.values(publicPaths)) {
  for (const [method, operation] of Object.entries(pathObj)) {
    if (method === 'parameters') continue
    if (Array.isArray(operation.tags)) {
      operation.tags.forEach(t => usedTags.add(t))
    }
  }
}

if (Array.isArray(doc.tags)) {
  doc.tags = doc.tags.filter(t => usedTags.has(t.name))
}

const yamlStr = yaml.dump(doc, { 
  lineWidth: -1,
  noRefs: true 
})

fs.writeFileSync(outputPath, yamlStr, 'utf8')
console.log(`Successfully generated public API spec with ${routeCount} public routes at ${outputPath}`)
