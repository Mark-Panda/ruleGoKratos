import { injectable, Emitter } from '@flowgram.ai/free-layout-editor';

@injectable()
export class DirtyService {
  private _dirty = false;

  private emitter = new Emitter<boolean>();

  public get dirty(): boolean {
    return this._dirty;
  }

  public onChange = this.emitter.event;

  public setDirty(val: boolean): void {
    this._dirty = !!val;
    this.emitter.fire(this._dirty);
  }
}
