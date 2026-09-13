import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import App from './App';

const rules = [
  {
    id: 'bath-rule',
    keywords: ['温泉', '入浴'],
    question: '温泉はどこですか？',
    answer: '温泉は2階にあります。',
    arActions: [{ type: 'POINT', target: 'stairs', params: { floor: '2' } }],
  },
];

beforeEach(() => {
  global.fetch = jest.fn().mockResolvedValue({
    ok: true,
    json: async () => rules,
  });
});

afterEach(() => {
  jest.restoreAllMocks();
});

test('loads rules and selects one for editing', async () => {
  render(<App />);

  const question = await screen.findByText('温泉はどこですか？');
  expect(global.fetch).toHaveBeenCalledWith('/rule/items');

  await userEvent.click(question);

  const form = screen.getByRole('form');
  expect(within(form).getByDisplayValue('bath-rule')).toBeInTheDocument();
  expect(within(form).getByDisplayValue('温泉, 入浴')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'ルールを更新' })).toBeEnabled();
  await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(2));
});

test('shows an API error without hiding the editor', async () => {
  global.fetch.mockResolvedValueOnce({ ok: false });

  render(<App />);

  expect(await screen.findByText('ルールの読み込みに失敗しました')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'ルールを作成' })).toBeEnabled();
});
