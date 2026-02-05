# Git Workflow (GitHub Flow + Semver Tags)

## ОБЯЗАТЕЛЬНОЕ ПРАВИЛО

При выполнении любых задач, связанных с изменением файлов проекта, ВСЕГДА следовать этому workflow.

## Структура веток

```txt
master                              ← основная ветка, всегда deployable
 ├── feature/...                  ← новая функциональность
 ├── bugfix/...                   ← исправления багов
 ├── docs/...                     ← документация
 ├── refactor/...                 ← рефакторинг
 ├── test/...                     ← тесты
 └── hotfix/...                   ← критические production fixes
```

- **`master`** — единственная постоянная ветка. Всегда стабильна и готова к деплою.
- **Feature branches** — короткоживущие ветки от `master`, мерджатся обратно в `master`.
- **Релизы** — git tags `vX.Y.Z` на `master`. CI собирает Docker-образ и создаёт GitHub Release.

## Workflow

### 1. Перед началом работы

```bash
git checkout master
git pull origin master

# Создать feature branch от master
git checkout -b <type>/<short-description>
```

**Branch naming:**

- `feature/` — новая функциональность
- `bugfix/` — исправления багов
- `docs/` — документация
- `refactor/` — рефакторинг без изменения функционала
- `test/` — добавление/улучшение тестов
- `hotfix/` — критические production fixes

**Примеры:**

- `feature/admin-auth-oauth2`
- `docs/auth-mechanics-documentation`
- `bugfix/storage-element-wal-race-condition`

### 2. Выполнение работы

- Делать изменения в созданной ветке
- Можно делать промежуточные commits при необходимости
- **Быстрые правки** (опечатки, мелкие фиксы) можно коммитить напрямую в `master`

### 3. По завершении задачи — предложить commit

**Спросить пользователя:**
> Работа завершена. Создать commit?

**Commit message format (Conventional Commits):**

```txt
<type>(<scope>): <subject>

[optional body]
```

**Types:**

- `feat`: новая функциональность
- `fix`: исправление бага
- `docs`: документация
- `style`: форматирование
- `refactor`: рефакторинг
- `test`: тесты
- `chore`: mastertenance

### 4. После commit — merge в master

**Спросить пользователя:**

> Commit создан. Выберите способ merge в `master`:
>
> **[A] Локальный merge:**
>
> ```bash
> git checkout master
> git merge --no-ff <branch-name>
> git push origin master
> ```
>
> **[B] GitHub PR:**
>
> ```bash
> git push origin <branch-name>
> gh pr create --base master --fill
> ```

### 5. После merge — удалить временную ветку

```bash
git branch -d <branch-name>
git push origin --delete <branch-name>
```

### 6. Выпуск релиза — создание тега

Когда набрана функциональность для релиза:

```bash
git checkout master
git pull origin master
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

CI автоматически:

- Собирает Docker-образ с тегом `vX.Y.Z`
- Создаёт GitHub Release

### 7. Поддержка старой версии (при необходимости)

Если нужно выпустить patch для старой версии:

```bash
# Создать release-ветку от тега
git checkout -b release/v0.1 v0.1.0

# Cherry-pick нужные фиксы
git cherry-pick <commit-hash>

# Выпустить patch
git tag -a v0.1.1 -m "Release v0.1.1"
git push origin v0.1.1
```

## Важные правила

1. **`master` всегда deployable** — не мерджить сломанный код
2. **Короткоживущие ветки** — merge как можно скорее
3. **Conventional Commits** — всегда использовать правильный формат
4. **Удалять ветки после merge** — не оставлять мусор
5. **Релизы через теги** — не через ветки
6. **PR для значимых изменений** — code review перед merge

## Пример полного цикла

```bash
# 1. Обновить master
git checkout master && git pull

# 2. Создать feature-ветку
git checkout -b docs/update-readme-authentication

# 3. Работа... (изменения файлов)

# 4. Commit
git add .
git commit -m "docs(admin-module): add authentication documentation"

# 5. Merge в master
git checkout master
git merge --no-ff docs/update-readme-authentication
git push origin master

# 6. Cleanup
git branch -d docs/update-readme-authentication

# 7. Когда готов релиз
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```
