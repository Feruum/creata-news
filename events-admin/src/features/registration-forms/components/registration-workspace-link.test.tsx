import assert from 'node:assert/strict'
import { test } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { RegistrationWorkspaceLink } from './registration-workspace-link'

test('registration workspace entry respects edit and registration management permissions', () => {
  for (const permissions of [
    { canEdit: true, canManageRegistrations: false },
    { canEdit: false, canManageRegistrations: true }
  ]) {
    const markup = renderToStaticMarkup(<RegistrationWorkspaceLink eventId={42} {...permissions} />)
    assert.match(markup, /href="\/event-registration\/42"/)
    assert.match(markup, /aria-label="Форма и ответы"/)
  }
  assert.equal(
    renderToStaticMarkup(<RegistrationWorkspaceLink eventId={42} canEdit={false} canManageRegistrations={false} />),
    ''
  )
})
