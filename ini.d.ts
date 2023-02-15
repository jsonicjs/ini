import { Jsonic } from '@jsonic/jsonic-next';
type IniOptions = {
    allowTrailingComma?: boolean;
    disallowComments?: boolean;
};
declare function Ini(jsonic: Jsonic, options: IniOptions): void;
export { Ini };
export type { IniOptions };
