# E2E Test Suite for QL-RF Control Tower

Comprehensive end-to-end test suite built with Playwright for the QuantumLayer Resilience Fabric Control Tower frontend.

## Test Coverage Summary

| Test File | Tests | Coverage Areas |
|-----------|-------|----------------|
| `accessibility.spec.ts` | 8 | WCAG compliance, landmarks, ARIA labels, keyboard navigation |
| `ai-copilot.spec.ts` | 7 | Legacy AI copilot basic tests |
| `ai.spec.ts` | 35 | AI chat, task submission, agents, task history, approvals |
| `compliance.spec.ts` | 33 | Compliance dashboard, frameworks, controls, image compliance |
| `drift.spec.ts` | 30 | Drift detection, visualization, remediation, filtering |
| `images.spec.ts` | 22 | Golden image management, lineage, versioning, filtering |
| `navigation.spec.ts` | 7 | Sidebar navigation, mobile menu, page routing |
| `overview.spec.ts` | 7 | Dashboard overview, metrics, widgets |
| `resilience.spec.ts` | 31 | DR readiness, pairs, failover tests, sync status |
| `settings.spec.ts` | 50 | Connectors, notifications, team, API keys, audit logs |

**Total Tests: 230**

## Test Structure

```
e2e/
├── fixtures/
│   ├── mock-data.ts      # Mock API responses for all pages
│   └── test-utils.ts     # Reusable test helper functions
├── accessibility.spec.ts  # Accessibility and a11y tests
├── ai.spec.ts            # AI Copilot comprehensive tests
├── compliance.spec.ts    # Compliance dashboard tests
├── drift.spec.ts         # Drift detection tests
├── images.spec.ts        # Golden image management tests
├── navigation.spec.ts    # Navigation and routing tests
├── overview.spec.ts      # Dashboard overview tests
├── resilience.spec.ts    # DR and resilience tests
└── settings.spec.ts      # Settings page tests
```

## Running Tests

### All Tests
```bash
npm run test:e2e
```

### Specific Browser
```bash
npm run test:e2e:chromium
npm run test:e2e -- --project=firefox
npm run test:e2e -- --project=webkit
```

### Interactive UI Mode
```bash
npm run test:e2e:ui
```

### Headed Mode (see browser)
```bash
npm run test:e2e:headed
```

### Specific Test File
```bash
npx playwright test e2e/images.spec.ts
npx playwright test e2e/compliance.spec.ts
```

### Specific Test
```bash
npx playwright test -g "should display page title"
npx playwright test compliance -g "frameworks"
```

### View Report
```bash
npm run test:e2e:report
```

## Test Categories

### 1. Page Structure Tests
- Page titles and descriptions
- Main content sections
- Navigation elements
- Action buttons

### 2. Data Display Tests
- Metric cards and statistics
- Tables and data grids
- Charts and visualizations
- Status badges and indicators

### 3. User Interaction Tests
- Form inputs and submissions
- Button clicks and actions
- Tab navigation
- Filtering and search
- Modal dialogs and dropdowns

### 4. API Integration Tests
- Data loading from APIs
- Loading states (skeletons)
- Error handling and retry
- Empty states

### 5. Accessibility Tests
- ARIA landmarks (main, nav, banner)
- Heading hierarchy
- Button and link labels
- Form labels
- Keyboard navigation
- Focus management

### 6. Responsive Tests
- Mobile viewport tests
- Tablet viewport tests
- Desktop viewport tests

## Mock Data

All tests use mock API responses defined in `fixtures/mock-data.ts`:

- **mockImages**: Golden image data with versions and lineage
- **mockDriftReport**: Drift detection data by platform and site
- **mockComplianceSummary**: Compliance frameworks and failing controls
- **mockResilienceSummary**: DR pairs and unpaired sites
- **mockAITasks**: AI task history and status
- **mockAIAgents**: AI agent information and statistics
- **mockSites**: Site information across platforms
- **mockRiskSummary**: Risk scoring data

## Test Utilities

`fixtures/test-utils.ts` provides helper functions:

### Navigation & Waiting
- `waitForPageReady()`: Wait for page load
- `waitForLoadingToFinish()`: Wait for loading indicators
- `waitAndClick()`: Wait for element then click

### API Mocking
- `mockAPIResponse()`: Mock single API endpoint
- `mockMultipleAPIs()`: Mock multiple endpoints
- `mockAPIWithDelay()`: Simulate slow API

### Accessibility
- `checkAccessibilityLandmarks()`: Verify ARIA landmarks
- `checkHeadingHierarchy()`: Check proper h1-h6 structure
- `checkTooltip()`: Verify tooltip on hover

### UI Checks
- `checkMetricCard()`: Verify metric card display
- `checkStatusBadge()`: Check status badge presence
- `checkTableHasData()`: Verify table has rows
- `checkEmptyState()`: Check empty state display
- `clickTab()`: Click tab and wait for activation

### Forms & Interaction
- `fillForm()`: Fill multiple form fields
- `pressKey()`: Press keyboard key
- `hoverElement()`: Hover over element

## Writing New Tests

### Basic Test Template
```typescript
import { test, expect } from "@playwright/test";
import { waitForPageReady, mockAPIResponse } from "./fixtures/test-utils";
import { mockData } from "./fixtures/mock-data";

test.describe("Feature Name", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/endpoint/, mockData);
    await page.goto("/feature");
    await waitForPageReady(page);
  });

  test("should display expected content", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Title" })).toBeVisible();
  });
});
```

### Adding Mock Data
Add to `fixtures/mock-data.ts`:
```typescript
export const mockNewFeature = {
  id: "feature-1",
  name: "Feature Name",
  status: "active",
  // ... other fields
};
```

### Best Practices

1. **Use data-testid sparingly**: Prefer semantic selectors (roles, labels)
2. **Mock API calls**: Don't rely on real backend in E2E tests
3. **Wait properly**: Use `waitForPageReady()` and `waitForLoadingToFinish()`
4. **Handle loading states**: Test both loading and loaded states
5. **Test error states**: Mock API failures to test error handling
6. **Check accessibility**: Include a11y checks in every page test
7. **Be defensive**: Use `.catch(() => false)` for optional elements
8. **Group related tests**: Use `test.describe()` blocks

## CI/CD Integration

Tests are configured for CI with:
- 2 retries on failure
- Screenshot on failure
- Video recording on failure
- JSON report output
- HTML report generation

### GitHub Actions Example
```yaml
- name: Install dependencies
  run: npm ci

- name: Install Playwright Browsers
  run: npx playwright install --with-deps

- name: Run E2E tests
  run: npm run test:e2e

- name: Upload test results
  if: always()
  uses: actions/upload-artifact@v3
  with:
    name: playwright-report
    path: playwright-report/
```

## Debugging Tests

### Debug Mode
```bash
npx playwright test --debug
```

### Trace Viewer
```bash
npx playwright show-trace trace.zip
```

### Inspect Test
```bash
npx playwright test --ui
```

### Console Logs
Set `DEBUG=pw:api` for verbose logging:
```bash
DEBUG=pw:api npm run test:e2e
```

## Browser Support

Tests run on:
- ✅ Chromium (Chrome, Edge)
- ✅ Firefox
- ✅ WebKit (Safari)
- ✅ Mobile Chrome (Pixel 5)
- ✅ Mobile Safari (iPhone 12)

## Performance

- **Parallel execution**: Tests run in parallel when possible
- **Fast feedback**: Average test duration ~2-5 seconds
- **Smart waits**: Automatic waiting for elements
- **Optimized selectors**: Efficient element location

## Known Issues & Limitations

1. **Auth**: Tests assume dev mode (no authentication required)
2. **Real-time features**: WebSocket tests not yet implemented
3. **File uploads**: File upload tests limited
4. **Third-party integrations**: Slack/Teams webhooks not tested

## Future Enhancements

- [ ] Visual regression testing
- [ ] Performance metrics collection
- [ ] Network throttling tests
- [ ] Internationalization (i18n) tests
- [ ] Cross-browser screenshot comparison
- [ ] API contract testing
- [ ] Load testing integration

## Contributing

When adding new features:

1. Add mock data to `fixtures/mock-data.ts`
2. Create test file: `e2e/feature-name.spec.ts`
3. Include test categories:
   - Page structure
   - Data display
   - User interactions
   - Loading/error states
   - Accessibility
4. Update this README with test count

## Support

For issues or questions:
- Check Playwright docs: https://playwright.dev
- Review test examples in existing spec files
- Ask in team chat or open GitHub issue
