export default async function teardown() {
  try {
    await fetch('http://127.0.0.1:6886/__stop', { method: 'POST' })
  } catch {
    // The test server may already have exited after a failed startup.
  }
}
