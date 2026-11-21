import { expect, test } from '@playwright/test'

test('opening the role list with a held pointer does not activate the user row', async ({ page }) => {
  await page.goto('/e2e/fixtures/user-roles.html')
  await page.getByRole('button', { name: 'Assign Role', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: 'Assign Role' })
  await dialog.getByRole('combobox', { name: 'Platform Role' }).click({ delay: 150 })
  await expect(page.getByTestId('opened-user')).toBeEmpty()
  await expect(dialog).toBeVisible()
  await page.getByRole('option', { name: 'Administrator', exact: true }).click()
  await dialog.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  await expect(page.getByRole('cell', { name: 'Admin', exact: true })).toBeVisible()
  await expect(page.getByTestId('opened-user')).toBeEmpty()
})

test('dismissing a role dialog through its backdrop does not open the user profile', async ({ page }) => {
  await page.goto('/e2e/fixtures/user-roles.html')
  await page.getByRole('button', { name: 'Assign Role', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: 'Assign Role' })
  await expect(dialog).toBeVisible()
  await page.mouse.click(10, 10)
  await expect(dialog).toHaveCount(0)
  await expect(page.getByTestId('opened-user')).toBeEmpty()
})

test('roles can be assigned with the keyboard without activating the user row', async ({ page }) => {
  await page.goto('/e2e/fixtures/user-roles.html')
  const trigger = page.getByRole('button', { name: 'Assign Role', exact: true })
  await trigger.focus()
  await page.keyboard.press('Enter')
  const dialog = page.getByRole('dialog', { name: 'Assign Role' })
  await dialog.getByRole('combobox', { name: 'Platform Role' }).focus()
  await page.keyboard.press('Space')
  await page.keyboard.press('End')
  await page.keyboard.press('Enter')
  await expect(dialog.getByRole('combobox', { name: 'Platform Role' })).toContainText('Administrator')
  await dialog.getByRole('button', { name: 'Save', exact: true }).focus()
  await page.keyboard.press('Enter')
  await expect(dialog).toHaveCount(0)
  await expect(page.getByRole('cell', { name: 'Admin', exact: true })).toBeVisible()
  await expect(page.getByTestId('opened-user')).toBeEmpty()
})

test('clicking a user row still opens the profile', async ({ page }) => {
  await page.goto('/e2e/fixtures/user-roles.html')
  await page.getByRole('cell', { name: 'role-user@example.test', exact: true }).click()
  await expect(page.getByTestId('opened-user')).toHaveText('5')
})

test('an administrator can assign a platform role from the user row', async ({ page }) => {
  await page.goto('/e2e/fixtures/user-roles.html')
  const row = page.getByRole('row').filter({ hasText: 'role-user@example.test' })
  await expect(row.getByRole('button', { name: 'Delete row' })).toHaveCount(0)
  await row.getByRole('button', { name: 'Assign Role', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: 'Assign Role' })
  await expect(dialog).toBeVisible()
  await expect(page.getByTestId('opened-user')).toBeEmpty()
  await dialog.getByRole('combobox', { name: 'Platform Role' }).click()
  await page.getByRole('option', { name: 'Administrator', exact: true }).click()
  await dialog.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  await expect(row.getByRole('cell', { name: 'Admin', exact: true })).toBeVisible()
  await expect(page.getByTestId('opened-user')).toBeEmpty()
})

test('a rejected assignment keeps the original role and the dialog open', async ({ page }) => {
  await page.goto('/e2e/fixtures/user-roles.html?denied=true')
  await page.getByRole('button', { name: 'Assign Role', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: 'Assign Role' })
  await dialog.getByRole('combobox', { name: 'Platform Role' }).click()
  await page.getByRole('option', { name: 'Administrator', exact: true }).click()
  await dialog.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByText(/Platform administrator access required/)).toBeVisible()
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button', { name: 'Cancel', exact: true }).click()
  await expect(
    page.getByRole('row').filter({ hasText: 'role-user@example.test' }).getByRole('cell', {
      name: 'User',
      exact: true
    })
  ).toBeVisible()
})

test('read-only tables hide row actions and selection', async ({ page }) => {
  await page.goto('/e2e/fixtures/user-roles.html?readOnly=true')
  await expect(page.getByRole('cell', { name: 'role-user@example.test' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Assign Role', exact: true })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Delete row' })).toHaveCount(0)
  await expect(page.getByRole('checkbox', { name: 'Select row' })).toHaveCount(0)
})

test('tables with deletion handlers retain their delete confirmation', async ({ page }) => {
  await page.goto('/e2e/fixtures/user-roles.html?deletion=true')
  await page.getByRole('button', { name: 'Delete row' }).click()
  const dialog = page.getByRole('alertdialog')
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button', { name: /Delete/ }).click()
  await expect(page.getByTestId('deleted-users')).toHaveText('[5]')
})
