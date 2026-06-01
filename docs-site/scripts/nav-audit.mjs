import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import assert from 'node:assert/strict'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const configPath = path.resolve(__dirname, '../.vitepress/config.js')

async function runAudit() {
  console.log('🔍 Running EntSaaS docs navbar & accessibility audit...')
  
  if (!fs.existsSync(configPath)) {
    console.error('❌ VitePress config.js not found!')
    process.exit(1)
  }

  // Dynamically import the VitePress config
  const module = await import(configPath)
  const config = module.default

  assert(config.themeConfig, 'themeConfig must be defined')
  const themeConfig = config.themeConfig

  // 1. Branding Assets
  assert(themeConfig.logo, 'themeConfig.logo must be defined')
  assert(typeof themeConfig.logo === 'object', 'themeConfig.logo must be an object with light/dark properties, not a plain string')
  assert(themeConfig.logo.light, 'themeConfig.logo.light must exist for positive navbar display')
  assert(themeConfig.logo.dark, 'themeConfig.logo.dark must exist for negative navbar display')
  
  // Must use EntSaaS logos
  assert(themeConfig.logo.light.includes('entsaas-'), 'themeConfig.logo.light must reference an EntSaaS brand asset')
  assert(themeConfig.logo.dark.includes('entsaas-'), 'themeConfig.logo.dark must reference an EntSaaS brand asset')
  
  // 2. Accessibility
  assert(themeConfig.logo.alt, 'themeConfig.logo.alt must exist for screen reader accessibility to the home link')

  assert(Array.isArray(themeConfig.nav), 'themeConfig.nav must be an array')
  assert(themeConfig.nav.length > 0, 'themeConfig.nav must have entries')

  // Verify all top-level nav elements
  for (const navItem of themeConfig.nav) {
    if (!navItem.link) continue
    
    assert(navItem.activeMatch, `nav item "${navItem.text}" must have an activeMatch defined for user context routing`)
    assert(navItem.ariaLabel, `nav item "${navItem.text}" must have an ariaLabel defined for screen reader support`)
  }

  // 3. SEO & Production Exposure Structure
  assert(config.titleTemplate === ':title | EntSaaS Docs', 'config.titleTemplate must be configured to append standard branding for SEO.')
  assert(config.metaChunk === true, 'config.metaChunk must be enabled for optimal CDN client caching')
  assert(Array.isArray(config.ignoreDeadLinks), 'config.ignoreDeadLinks must be an array to whitelist dev targets')
  assert(config.ignoreDeadLinks.length > 0 && config.ignoreDeadLinks[0].toString().includes('localhost'), 'config.ignoreDeadLinks must whitelist only localhost dev routes explicitly.')
  
  assert(config.sitemap, 'config.sitemap must be defined so bots can build indexing maps.')
  assert(config.sitemap.hostname === 'https://docs.entsaas.com', 'config.sitemap.hostname must point strictly to https://docs.entsaas.com.')
  assert(typeof config.transformHead === 'function', 'config.transformHead must be defined to inject canonical links.')

  console.log('✅ Docs navbar audit PASSED. Branding and accessibility constraints verified.')
}

runAudit().catch(err => {
  console.error('\n❌ Navbar audit FAILED!')
  console.error(err.message)
  process.exit(1)
})
