import path from 'node:path'
import { fileURLToPath } from 'node:url'

import {
  buildCorrespondingSource,
  renderBuilding,
} from './corresponding-source-builder.mjs'
import {
  assertFullCommit,
  assertReleaseTag,
  parseGoDownload,
} from './release-policy.mjs'

export { buildCorrespondingSource, renderBuilding }
export {
  escapeModulePath,
  verifyCorrespondingSourceArchive,
  verifyCorrespondingSourceTree,
} from './corresponding-source-verifier.mjs'
export { parseGoDownload }
export {
  makeTreeWritable,
  renderTrackedFiles,
  sha256,
  treeDigest,
} from './tree-integrity.mjs'

function parseArguments(argv) {
  const values = {}
  for (let index = 0; index < argv.length; index += 2) {
    const key = argv[index]
    const value = argv[index + 1]
    if (!key?.startsWith('--') || value === undefined) {
      throw new Error(
        'usage: build-corresponding-source.mjs --tag <tag> --commit <sha> --output <archive>',
      )
    }
    values[key.slice(2)] = value
  }
  assertReleaseTag(values.tag)
  assertFullCommit(values.commit)
  if (!values.output) throw new Error('--output is required')
  return values
}

const isCLI =
  process.argv[1] &&
  path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)

if (isCLI) {
  try {
    const values = parseArguments(process.argv.slice(2))
    await buildCorrespondingSource({
      root: path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..'),
      tag: values.tag,
      commit: values.commit,
      output: values.output,
    })
  } catch (error) {
    console.error(error.message)
    process.exitCode = 1
  }
}
