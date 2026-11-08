"""Auditoría estática: reglas de seguridad que el código debe cumplir siempre."""

import ast
import re
from pathlib import Path

from fps_booster.context import MODE_SIMULATION, build_context
from fps_booster.system.backend import SimulatedSystemBackend, WindowsSystemBackend

SRC = Path(__file__).resolve().parents[1] / "src" / "fps_booster"
SOURCES = {p: p.read_text(encoding="utf-8") for p in SRC.rglob("*.py")}


def test_no_process_termination_anywhere():
    pattern = re.compile(r"\.(kill|terminate|suspend)\s*\(|taskkill|Stop-Process|TerminateProcess")
    offenders = [str(p.relative_to(SRC)) for p, text in SOURCES.items() if pattern.search(text)]
    assert offenders == [], f"Código que cierra procesos: {offenders}"


def test_no_shell_true_or_os_system():
    offenders = [str(p.relative_to(SRC)) for p, t in SOURCES.items()
                 if "shell=True" in t or "os.system(" in t or "os.popen(" in t]
    assert offenders == []


def test_registry_writes_only_in_whitelisted_module():
    for path, text in SOURCES.items():
        if path.name == "registry.py":
            continue
        assert "SetValueEx" not in text and "DeleteValue" not in text, path


def test_real_backend_only_instantiated_in_context():
    users = []
    for path, text in SOURCES.items():
        tree = ast.parse(text)
        for node in ast.walk(tree):
            if isinstance(node, ast.Call) and getattr(node.func, "id", None) == "WindowsSystemBackend":
                users.append(path.name)
    assert users == ["context.py"]


def test_default_context_is_simulation(tmp_path):
    ctx = build_context(data_dir=tmp_path)
    assert ctx.mode == MODE_SIMULATION
    assert isinstance(ctx.backend, SimulatedSystemBackend)
    assert not isinstance(ctx.backend, WindowsSystemBackend)
    assert ctx.engine.simulated
    assert "simulacion" in ctx.backups.path.name  # backups separados del modo real


def test_every_optimization_is_reversible_and_low_risk():
    from fps_booster.optimizations.catalog import default_catalog

    for opt in default_catalog().values():
        assert opt.reversible, opt.id
        assert opt.requires_admin is False, opt.id
        assert opt.risk in ("bajo", "medio"), opt.id


def test_consent_only_created_by_dialog_or_explicit_question():
    """En la UI, Consent.grant solo aparece en el diálogo de consentimiento y en la confirmación de revertir."""
    allowed = {"consent_dialog.py", "main_window.py", "engine.py"}
    users = {p.name for p, t in SOURCES.items() if "Consent.grant(" in t or "Consent(" in t}
    assert users <= allowed, users
    main = SOURCES[SRC / "ui" / "main_window.py"]
    assert main.count("Consent.grant(") == 1  # solo en consent_for_revert, tras QMessageBox.question
