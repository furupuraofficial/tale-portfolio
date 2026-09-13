import { useCallback, useEffect, useMemo, useState } from 'react';
import './App.css';

const rawBase = import.meta.env.VITE_API_BASE || '';
const API_BASE = rawBase.replace(/\/$/, '');
const withBase = (path) => (API_BASE ? `${API_BASE}${path}` : path);
const createEmptyAction = () => ({ type: '', target: '', paramsText: '{}' });
const emptyForm = () => ({
  id: '',
  keywords: '',
  question: '',
  answer: '',
  arActions: [createEmptyAction()],
});

function App() {
  const [rules, setRules] = useState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const [selectedId, setSelectedId] = useState('');
  const [form, setForm] = useState(() => emptyForm());
  const [search, setSearch] = useState('');

  const filteredRules = useMemo(() => {
    if (!search.trim()) return rules;
    const query = search.toLowerCase();
    return rules.filter(
      (rule) =>
        rule.question.toLowerCase().includes(query) ||
        rule.answer.toLowerCase().includes(query) ||
        rule.keywords.some((kw) => kw.toLowerCase().includes(query))
    );
  }, [rules, search]);

  const resetForm = useCallback(() => {
    setSelectedId('');
    setForm(emptyForm());
  }, []);

  const selectRule = useCallback((rule, { clearFeedback = true } = {}) => {
    setSelectedId(rule.id);
    setForm({
      id: rule.id,
      keywords: rule.keywords.join(', '),
      question: rule.question,
      answer: rule.answer,
      arActions:
        rule.arActions && rule.arActions.length
          ? rule.arActions.map((action) => ({
              type: action.type || '',
              target: action.target || '',
              paramsText: action.params ? JSON.stringify(action.params) : '{}',
            }))
          : [createEmptyAction()],
    });
    if (clearFeedback) {
      setStatus('');
      setError('');
    }
  }, []);

  const loadRules = useCallback(async (selectionToPreserve = '') => {
    try {
      const res = await fetch(withBase('/rule/items'));
      if (!res.ok) {
        throw new Error('ルールの読み込みに失敗しました');
      }
      const data = await res.json();
      setRules(data);
      if (selectionToPreserve) {
        const match = data.find((r) => r.id === selectionToPreserve);
        if (match) {
          selectRule(match, { clearFeedback: false });
        } else {
          resetForm();
        }
      }
    } catch (err) {
      setError(err.message || 'ルールを読み込めませんでした');
    } finally {
      setLoading(false);
    }
  }, [selectRule, resetForm]);

  useEffect(() => {
    // The rule list is external state and must be synchronized on first mount.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadRules();
  }, [loadRules]);

  function reloadRules() {
    setLoading(true);
    setError('');
    loadRules(selectedId);
  }

  function updateField(field, value) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  function updateAction(index, field, value) {
    setForm((prev) => {
      const updated = [...prev.arActions];
      updated[index] = { ...updated[index], [field]: value };
      return { ...prev, arActions: updated };
    });
  }

  function addAction() {
    setForm((prev) => ({ ...prev, arActions: [...prev.arActions, createEmptyAction()] }));
  }

  function removeAction(index) {
    setForm((prev) => {
      if (prev.arActions.length === 1) return prev;
      const updated = prev.arActions.filter((_, i) => i !== index);
      return { ...prev, arActions: updated };
    });
  }

  async function handleSave(e) {
    e.preventDefault();
    setSaving(true);
    setError('');
    setStatus('');

    const keywords = form.keywords
      .split(',')
      .map((k) => k.trim())
      .filter(Boolean);

    const arActions = [];
    for (const action of form.arActions) {
      if (!action.type.trim()) continue;
      let params = undefined;
      if (action.paramsText.trim()) {
        try {
          params = JSON.parse(action.paramsText);
        } catch {
          setSaving(false);
          setError('AR Actionのパラメーターは有効なJSONで入力してください。');
          return;
        }
      }
      arActions.push({
        type: action.type.trim(),
        target: action.target.trim(),
        params,
      });
    }

    const payload = {
      id: form.id.trim() || undefined,
      keywords,
      question: form.question.trim(),
      answer: form.answer.trim(),
      arActions,
    };

    try {
      const newId = form.id.trim();

      // IDを変更する場合は新規作成->旧IDを削除の順で対応
      if (selectedId && newId && newId !== selectedId) {
        const createRes = await fetch(withBase('/rule/items'), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ ...payload, id: newId }),
        });
        if (!createRes.ok) {
          const message = await createRes.text();
          throw new Error(message || '保存に失敗しました');
        }

        const deleteRes = await fetch(withBase(`/rule/items/${selectedId}`), { method: 'DELETE' });
        if (!deleteRes.ok) {
          throw new Error('新しいIDで作成しましたが、旧ルールの削除に失敗しました。');
        }

        setStatus('ルールIDを変更して保存しました');
      } else {
        const res = await fetch(
          selectedId ? withBase(`/rule/items/${selectedId}`) : withBase('/rule/items'),
          {
            method: selectedId ? 'PUT' : 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
          }
        );

        if (!res.ok) {
          const message = await res.text();
          throw new Error(message || '保存に失敗しました');
        }

        setStatus(selectedId ? 'ルールを更新しました' : 'ルールを作成しました');
      }

      await loadRules(newId || selectedId);
      if (!selectedId) {
        resetForm();
      } else if (newId && newId !== selectedId) {
        setSelectedId(newId);
      }
    } catch (err) {
      setError(err.message || '保存に失敗しました');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id) {
    if (!id) return;
    const confirm = window.confirm('このルールを削除しますか？');
    if (!confirm) return;
    setSaving(true);
    setError('');
    setStatus('');
    try {
      const res = await fetch(withBase(`/rule/items/${id}`), { method: 'DELETE' });
      if (!res.ok) {
        throw new Error('削除に失敗しました');
      }
      setStatus('ルールを削除しました');
      await loadRules();
      resetForm();
    } catch (err) {
      setError(err.message || '削除に失敗しました');
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page">
      <header className="hero">
        <div>
          <p className="eyebrow">ルール管理パネル</p>
          <h1>スクリプト返信とARアクションを管理</h1>
          <p className="lede">会話エンジンで使うルールを読み込み、追加・編集・削除できます。変更はすべて即座にバックエンドAPIへ反映されます。</p>
          <div className="hero-actions">
            <button className="ghost" onClick={reloadRules} disabled={loading}>
              {loading ? '再読み込み中…' : '再読み込み'}
            </button>
            <button className="ghost" onClick={resetForm}>
              新規ルール
            </button>
          </div>
        </div>
        <div className="badge">
          <div className="badge-dot" />
          <div>
            <p className="badge-label">APIエンドポイント</p>
            <p className="badge-value">{withBase('/rule/items')}</p>
          </div>
        </div>
      </header>

      <main className="grid">
        <section className="panel">
          <div className="panel-header">
            <div>
              <p className="eyebrow">ルール一覧</p>
              <h2>登録済みルール</h2>
            </div>
            <input
              className="search"
              placeholder="キーワードまたは質問で検索"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>

          {loading ? (
            <p className="muted">ルールを読み込み中…</p>
          ) : filteredRules.length === 0 ? (
            <p className="muted">該当するルールがありません。検索条件を変えるか、新しく作成してください。</p>
          ) : (
            <div className="rule-list">
              {filteredRules.map((rule) => (
                <button
                  key={rule.id}
                  className={`rule-card ${selectedId === rule.id ? 'active' : ''}`}
                  onClick={() => selectRule(rule)}
                >
                  <div className="rule-meta">
                    <span className="pill">{rule.id}</span>
                    <span className="pill subtle">{rule.keywords.join(', ') || 'キーワードなし'}</span>
                  </div>
                  <p className="rule-question">{rule.question || '質問が未設定'}</p>
                  <p className="rule-answer">{rule.answer || '回答がまだ設定されていません。'}</p>
                </button>
              ))}
            </div>
          )}
        </section>

        <section className="panel">
          <div className="panel-header">
            <div>
              <p className="eyebrow">{selectedId ? 'ルールを編集' : 'ルールを作成'}</p>
              <h2>{selectedId || '新規ルール'}</h2>
            </div>
            {selectedId && (
              <button className="danger ghost" onClick={() => handleDelete(selectedId)} disabled={saving}>
                削除
              </button>
            )}
          </div>
          <form className="form" aria-label="ルール編集フォーム" onSubmit={handleSave}>
            <label>
              <span>ルールID（編集可・未入力なら自動生成）</span>
              <input
                type="text"
                value={form.id}
                onChange={(e) => updateField('id', e.target.value)}
                placeholder="rule-123"
              />
            </label>

            <label>
              <span>キーワード（カンマ区切り）</span>
              <input
                type="text"
                value={form.keywords}
                onChange={(e) => updateField('keywords', e.target.value)}
                placeholder="温泉, 露天風呂, スパ"
                required
              />
            </label>

            <label>
              <span>質問</span>
              <input
                type="text"
                value={form.question}
                onChange={(e) => updateField('question', e.target.value)}
                placeholder="温泉はどこにありますか？"
                required
              />
            </label>

            <label>
              <span>回答</span>
              <textarea
                value={form.answer}
                onChange={(e) => updateField('answer', e.target.value)}
                placeholder="温泉は2階にあります…"
                rows={4}
                required
              />
            </label>

            <div className="actions-header">
              <div>
                <span>ARアクション</span>
                <p className="muted small">任意。タイプが空欄ならスキップされます。</p>
              </div>
              <button type="button" className="ghost" onClick={addAction}>
                アクションを追加
              </button>
            </div>

            {form.arActions.map((action, idx) => (
              <div className="action-row" key={idx}>
                <div className="action-grid">
                  <label>
                    <span>タイプ</span>
                    <input
                      type="text"
                      value={action.type}
                      onChange={(e) => updateAction(idx, 'type', e.target.value)}
                      placeholder="PLAY_ANIMATION"
                    />
                  </label>
                  <label>
                    <span>ターゲット</span>
                    <input
                      type="text"
                      value={action.target}
                      onChange={(e) => updateAction(idx, 'target', e.target.value)}
                      placeholder="zashiki"
                    />
                  </label>
                </div>
                <label>
                  <span>パラメーター（JSON）</span>
                  <textarea
                    value={action.paramsText}
                    onChange={(e) => updateAction(idx, 'paramsText', e.target.value)}
                    rows={3}
                    placeholder='{"name": "wave"}'
                  />
                </label>
                <div className="action-controls">
                  <button
                    type="button"
                    className="ghost danger"
                    onClick={() => removeAction(idx)}
                    disabled={form.arActions.length === 1}
                  >
                    削除
                  </button>
                </div>
              </div>
            ))}

            <div className="form-footer">
              <div className="messages">
                {error && <p className="error">{error}</p>}
                {status && <p className="status">{status}</p>}
              </div>
              <button type="submit" className="primary" disabled={saving}>
                {saving ? '保存中…' : selectedId ? 'ルールを更新' : 'ルールを作成'}
              </button>
            </div>
          </form>
        </section>
      </main>
    </div>
  );
}

export default App;
