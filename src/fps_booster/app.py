"""Punto de entrada de FPS BOOSTER."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

from fps_booster import APP_NAME, __version__


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(prog="fps-booster", description=f"{APP_NAME} {__version__}")
    parser.add_argument(
        "--modo-real", action="store_true",
        help="Permite aplicar cambios reales en Windows (pide confirmación). Por defecto: simulación.",
    )
    parser.add_argument("--data-dir", type=Path, default=None, help="Carpeta de datos alternativa.")
    parser.add_argument("--screenshot", type=Path, default=None,
                        help="Captura la ventana tras unos segundos y sale (para QA).")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)

    from PySide6.QtCore import QTimer
    from PySide6.QtWidgets import QApplication, QMessageBox

    from fps_booster.context import build_context
    from fps_booster.core.logging_setup import setup_logging
    from fps_booster.core.paths import data_dir
    from fps_booster.ui import theme
    from fps_booster.ui.main_window import MainWindow

    base = data_dir(args.data_dir)
    log = setup_logging(base / "logs")
    log.info("Iniciando %s %s", APP_NAME, __version__)

    app = QApplication.instance() or QApplication(sys.argv)
    app.setApplicationName(APP_NAME)
    app.setStyle("Fusion")
    app.setStyleSheet(theme.QSS)

    real = False
    if args.modo_real:
        answer = QMessageBox.warning(
            None, f"{APP_NAME} — Modo real",
            "Vas a iniciar en MODO REAL: los cambios que autorices modificarán la configuración "
            "de Windows (plan de energía y ajustes de usuario en el registro).\n\n"
            "Se guardan los valores originales para revertirlos. ¿Continuar en modo real?",
            QMessageBox.StandardButton.Yes | QMessageBox.StandardButton.No,
            QMessageBox.StandardButton.No,
        )
        real = answer == QMessageBox.StandardButton.Yes
        if not real:
            log.info("Modo real rechazado por el usuario: se usa simulación")

    ctx = build_context(real_mode=real, data_dir=base)
    log.info("Modo: %s", ctx.mode)
    window = MainWindow(ctx)
    window.show()

    if args.screenshot:
        def _capture() -> None:
            window.grab().save(str(args.screenshot))
            window.close()

        QTimer.singleShot(6000, _capture)

    try:
        return app.exec()
    finally:
        window.stop_workers()


if __name__ == "__main__":
    raise SystemExit(main())
